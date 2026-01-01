package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"

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
	mux := http.NewServeMux()
	h := newHandlers(shortener)

	mux.HandleFunc("/", h.shorten)
	mux.HandleFunc("/{id}", h.redirect)

	return mux
}

func (h *handlers) shorten(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	defer req.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(res, req.Body, 1024))
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
		http.Error(res, "Internal error", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	if _, err := io.WriteString(res, shortURL); err != nil {
		http.Error(res, "Internal error", http.StatusBadRequest)
		return
	}
}

func (h *handlers) redirect(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	defer req.Body.Close()
	if _, err := io.Copy(io.Discard, http.MaxBytesReader(res, req.Body, 1024)); err != nil {
		http.Error(res, "Internal error", http.StatusBadRequest)
		return
	}

	originalURL, err := h.shortener.Expand(req.PathValue("id"))
	if errors.Is(err, service.ErrTokenNotFound) {
		http.Error(res, "No such URL", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
