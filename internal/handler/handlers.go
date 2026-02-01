package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
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

	r.Use(middleware.Logger)

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
