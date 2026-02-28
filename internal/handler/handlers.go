package handler

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/svnnj/shortener/internal/model"
	"github.com/svnnj/shortener/internal/service"
)

type handlers struct {
	shortener     service.ShortenerService
	healthChecker service.HealthCheckerService
}

func newHandlers(shortener service.ShortenerService, healthChecker service.HealthCheckerService) *handlers {
	return &handlers{
		shortener:     shortener,
		healthChecker: healthChecker,
	}
}

func NewRouter(shortener service.ShortenerService, healthChecker service.HealthCheckerService) http.Handler {
	h := newHandlers(shortener, healthChecker)
	r := chi.NewRouter()

	r.Use(withLogging)
	r.Use(withDecompression)
	r.Use(withCompression)
	r.Use(middleware.Recoverer)

	r.Post("/", h.shorten)
	r.Post("/api/shorten", h.shortenJSON)
	r.Post("/api/shorten/batch", h.shortenBatch)
	r.Get("/{id}", h.redirect)
	r.Get("/ping", h.ping)

	return r
}

func (h *handlers) shortenBatch(res http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	maxBytes := int64(1024)
	body, err := io.ReadAll(http.MaxBytesReader(res, req.Body, maxBytes))
	if err != nil {
		slog.Warn("failed to read request body", "error", err, "max_bytes", maxBytes)
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	var reqData []model.ShortenBatchReq
	if err := json.Unmarshal([]byte(body), &reqData); err != nil {
		slog.Warn("failed to parse JSON", "error", err, "body", string(body))
		http.Error(res, "Error parsing JSON", http.StatusBadRequest)
		return
	}
	if len(reqData) == 0 {
		slog.Warn("empty batch request received")
		http.Error(res, "Empty array", http.StatusBadRequest)
		return
	}

	for _, v := range reqData {
		parsedURL, err := url.Parse(v.OriginalURL)
		if err != nil || parsedURL.Host == "" {
			slog.Warn("invalid URL in batch", "url", v.OriginalURL, "error", err)
			http.Error(res, fmt.Sprintf("Incorrect URL: %s", v.OriginalURL), http.StatusBadRequest)
			return
		}
	}

	resData, err := h.shortener.ShortenBatch(req.Context(), reqData)
	if err != nil {
		slog.Error("failed to shorten batch", "error", err, "batch_size", len(reqData))
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	var resJSON []byte
	resJSON, err = json.Marshal(resData)
	if err != nil {
		slog.Error("failed to marshal response", "error", err, "response_data", resData)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := res.Write(resJSON); err != nil {
		slog.Error("failed to write response", "error", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handlers) shortenJSON(res http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	maxBytes := int64(1024)
	body, err := io.ReadAll(http.MaxBytesReader(res, req.Body, maxBytes))
	if err != nil {
		slog.Warn("failed to read request body", "error", err, "max_bytes", maxBytes)
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	var reqData model.ShortenJSONReq
	if err := json.Unmarshal([]byte(body), &reqData); err != nil {
		slog.Warn("failed to parse JSON", "error", err, "body", string(body))
		http.Error(res, "Error parsing JSON", http.StatusBadRequest)
		return
	}
	parsedURL, err := url.Parse(reqData.URL)
	if err != nil || parsedURL.Host == "" {
		slog.Warn("invalid URL provided", "url", reqData.URL, "error", err)
		http.Error(res, "Incorrect URL", http.StatusBadRequest)
		return
	}

	var resData model.ShortenJSONRes
	shortURL, err := h.shortener.Shorten(req.Context(), reqData.URL)
	if err != nil {
		slog.Error("failed to shorten URL", "error", err, "original_url", reqData.URL)
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}
	resData.Result = shortURL

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	resJSON, err := json.Marshal(resData)
	if err != nil {
		slog.Error("failed to marshal response", "error", err, "response_data", resData)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := res.Write(resJSON); err != nil {
		slog.Error("failed to write response", "error", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handlers) shorten(res http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	maxBytes := int64(1024)
	body, err := io.ReadAll(http.MaxBytesReader(res, req.Body, maxBytes))
	if err != nil {
		slog.Warn("failed to read request body", "error", err, "max_bytes", maxBytes)
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	originalURL, err := url.Parse(string(body))
	if err != nil || originalURL.Host == "" {
		slog.Warn("invalid URL provided", "url", string(body), "error", err)
		http.Error(res, "Incorrect URL", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortener.Shorten(req.Context(), originalURL.String())
	if err != nil {
		slog.Error("failed to shorten URL", "error", err, "original_url", originalURL.String())
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(res, shortURL); err != nil {
		slog.Error("failed to write response", "error", err)
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

	token := chi.URLParam(req, "id")
	originalURL, err := h.shortener.Expand(req.Context(), token)
	if errors.Is(err, service.ErrTokenNotFound) {
		slog.Info("token not found", "token", token)
		http.Error(res, "No such URL", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("failed to expand token", "error", err, "token", token)
		http.Error(res, "Internal error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *handlers) ping(res http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(req.Body, 1024)); err != nil {
		slog.Warn("failed to discard request body", "error", err)
		http.Error(res, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.healthChecker.Ping(req.Context())
	if err != nil {
		if errors.Is(err, service.ErrNilConnection) {
			http.Error(res, "Database is not available", http.StatusInternalServerError)
			return
		}
		slog.Error("health check failed", "error", err)
		http.Error(res, "Internal server error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
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

func withDecompression(next http.Handler) http.Handler {
	compFn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		defer r.Body.Close()
		gzr, err := gzip.NewReader(r.Body)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		r.Body = gzr

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(compFn)
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func withCompression(next http.Handler) http.Handler {
	compFn := func(w http.ResponseWriter, r *http.Request) {
		isToBeCompressed := false
		for _, s := range r.Header.Values("Accept-Encoding") {
			if strings.Contains(s, "gzip") {
				isToBeCompressed = true
				break
			}
		}
		if strings.Contains(w.Header().Get("Content-Type"), "application/json") && strings.Contains(w.Header().Get("Content-Type"), "text/plain") {
			isToBeCompressed = false
		}
		if !isToBeCompressed {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	}

	return http.HandlerFunc(compFn)
}
