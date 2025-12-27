package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

type store struct {
	mu   sync.RWMutex
	data map[string]string
}

func newStore() *store {
	return &store{
		mu: sync.RWMutex{},
		data: map[string]string{
			"test1": "test2",
		},
	}
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *store) shortenHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(res, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(req.Body, 1024))
		if err != nil {
			http.Error(res, "Invalid request body", http.StatusBadRequest)
			return
		}
		req.Body.Close()

		originalURL := strings.TrimSpace(string(body))
		if len(originalURL) == 0 {
			http.Error(res, "Request body must not be empty", http.StatusBadRequest)
			return
		}

		var token string
		for i := 0; true; i++ {
			if token, err = randomToken(9); err != nil {
				http.Error(res, "Internal error", http.StatusBadRequest)
				return
			}
			s.mu.RLock()
			_, exists := s.data[token]
			s.mu.RUnlock()
			if !exists {
				break
			}
			if i >= 3 {
				http.Error(res, "Internal error", http.StatusBadRequest)
				return
			}
		}
		s.mu.Lock()
		s.data[token] = originalURL
		s.mu.Unlock()

		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(res, "http://"+req.Host+"/"+token); err != nil {
			http.Error(res, "Internal error", http.StatusBadRequest)
			return
		}
	}
}

func (s *store) redirectHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(res, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		if _, err := io.Copy(io.Discard, req.Body); err != nil {
			http.Error(res, "Internal error", http.StatusBadRequest)
			return
		}
		req.Body.Close()

		s.mu.RLock()
		originalURL, exists := s.data[req.PathValue("id")]
		s.mu.RUnlock()
		if !exists {
			http.Error(res, "No such URL", http.StatusBadRequest)
			return
		}

		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Location", originalURL)
		res.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func run() error {
	s := newStore()
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.shortenHandler())
	mux.HandleFunc("/{id}", s.redirectHandler())

	err := http.ListenAndServe("localhost:8081", mux)
	return err

}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
