package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi"
	"github.com/svnnj/shortener/internal/service"
)

type handlers struct {
	shortener service.ShortenerService
}

func newHandlers(shortener service.ShortenerService) *handlers {
	return &handlers{
		shortener: shortener,
	}
}

func NewRouter(shortener service.ShortenerService) http.Handler {
	h := newHandlers(shortener)
	r := chi.NewRouter()

	r.Use(withLogging)

	r.Post("/", h.shorten)
	r.Get("/{id}", h.redirect)

	return r
}

func (h *handlers) shorten(res http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	maxBytes := int64(1024)
	body, err := io.ReadAll(http.MaxBytesReader(res, req.Body, maxBytes))
	if err != nil {
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	originalURL, err := url.Parse(string(body))
	if err != nil || originalURL.Host == "" {
		http.Error(res, "Incorrect URL", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortener.Shorten(originalURL.String())
	if err != nil {
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(res, shortURL); err != nil {
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}
}

func (h *handlers) redirect(res http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(req.Body, 1024)); err != nil {
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	id := chi.URLParam(req, "id")
	originalURL, err := h.shortener.Expand(id)
	if errors.Is(err, service.ErrTokenNotFound) {
		http.Error(res, "No such URL", http.StatusNotFound)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func withLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw := loggingResponseWriter{
			w,
			&responseData{
				status: 0,
				size:   0,
			},
		}

		h.ServeHTTP(lw, r)

		duration := time.Since(start)

		slog.Info("REQ",
			"uri", r.RequestURI,
			"method", r.Method,
			"status", lw.responseData.status,
			"duration", duration,
			"size", lw.responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}
