package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/svnnj/shortener/internal/service"
)

type testTable []struct {
	name            string
	method          string
	path            string
	mock            shortenerMock
	body            string
	healthCheckMock healthCheckMock
	wantStatus      int
	wantCT          string
	wantLocation    string
	wantBody        string
}

type shortenerMock struct {
	shortenRet string
	shortenErr error

	expandRet string
	expandErr error
}

type healthCheckMock struct {
	pingErr error
}

func (s *shortenerMock) Shorten(originalURL string) (string, error) {
	return s.shortenRet, s.shortenErr
}

func (s *shortenerMock) Expand(token string) (string, error) {
	return s.expandRet, s.expandErr
}

func (hc *healthCheckMock) PingDB(ctx context.Context) error {
	return hc.pingErr
}

func TestHealthChecker_ping(t *testing.T) {
	tests := testTable{
		{
			name:            "successful GET",
			method:          http.MethodGet,
			healthCheckMock: healthCheckMock{},
			wantStatus:      http.StatusOK,
			wantCT:          "text/plain",
		},
		{
			name:            "reject non‑GET",
			method:          http.MethodPost,
			body:            "",
			healthCheckMock: healthCheckMock{},
			wantStatus:      http.StatusMethodNotAllowed,
		},
		{
			name:   "service error",
			method: http.MethodGet,
			healthCheckMock: healthCheckMock{
				pingErr: errors.New("boom"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Database is not available\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(&tt.mock, &tt.healthCheckMock)

			req := httptest.NewRequest(tt.method, "/ping", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantCT, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.wantBody, string(b))
		})
	}
}

func TestShorten_shorten(t *testing.T) {
	tests := testTable{
		{
			name:   "successful POST",
			method: http.MethodPost,
			body:   "https://example.com/foo",
			mock: shortenerMock{
				shortenRet: "http://test.test/abc123",
			},
			wantStatus: http.StatusCreated,
			wantCT:     "text/plain",
			wantBody:   "http://test.test/abc123",
		},
		{
			name:       "reject non‑POST",
			method:     http.MethodGet,
			body:       "",
			mock:       shortenerMock{},
			wantStatus: http.StatusMethodNotAllowed,
			wantCT:     "",
			wantBody:   "",
		},
		{
			name:       "invalid URL",
			method:     http.MethodPost,
			body:       "not-a-url",
			mock:       shortenerMock{},
			wantStatus: http.StatusBadRequest,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Incorrect URL\n",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			body:   "https://example.com",
			mock: shortenerMock{
				shortenErr: errors.New("boom"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Internal error\n",
		},
		{
			name:       "exceeds the limit",
			method:     http.MethodPost,
			body:       func() string { return "https://example.com/foo" + strings.Repeat("a", 2*1024) }(),
			mock:       shortenerMock{},
			wantStatus: http.StatusBadRequest,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Invalid request body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(&tt.mock, &tt.healthCheckMock)

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantCT, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.wantBody, string(b))
		})
	}
}

func TestShorten_shortenJSON(t *testing.T) {
	tests := testTable{
		{
			name:   "successful POST",
			method: http.MethodPost,
			body:   `{"url":"https://example.com/foo"}`,
			mock: shortenerMock{
				shortenRet: "http://test.test/abc123",
			},
			wantStatus: http.StatusCreated,
			wantCT:     "application/json",
			wantBody:   `{"result":"http://test.test/abc123"}`,
		},
		{
			name:       "reject non‑POST",
			method:     http.MethodGet,
			body:       "",
			mock:       shortenerMock{},
			wantStatus: http.StatusMethodNotAllowed,
			wantCT:     "",
			wantBody:   "",
		},
		{
			name:       "invalid URL",
			method:     http.MethodPost,
			body:       `{"url":"not-a-url"}`,
			mock:       shortenerMock{},
			wantStatus: http.StatusBadRequest,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Incorrect URL\n",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			body:   `{"url":"https://example.com"}`,
			mock: shortenerMock{
				shortenErr: errors.New("boom"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Internal error\n",
		},
		{
			name:       "exceeds the limit",
			method:     http.MethodPost,
			body:       func() string { return `"url":"https://example.com/foo"` + strings.Repeat("a", 2*1024) }(),
			mock:       shortenerMock{},
			wantStatus: http.StatusBadRequest,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "Invalid request body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(&tt.mock, &tt.healthCheckMock)

			req := httptest.NewRequest(tt.method, "/api/shorten", strings.NewReader(tt.body))
			req.Header.Add("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantCT, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.wantBody, string(b))
		})
	}
}

func TestShorten_redirect(t *testing.T) {
	tests := testTable{
		{
			name:   "succesful GET",
			method: http.MethodGet,
			path:   "/abc123",
			mock: shortenerMock{
				expandRet: "https://example.com/original",
			},
			wantStatus:   http.StatusTemporaryRedirect,
			wantCT:       "text/plain",
			wantLocation: "https://example.com/original",
			wantBody:     "",
		},
		{
			name:       "reject non‑GET",
			method:     http.MethodPost,
			path:       "/abc123",
			mock:       shortenerMock{},
			wantStatus: http.StatusMethodNotAllowed,
			wantCT:     "",
			wantBody:   "",
		},
		{
			name:   "token not found",
			method: http.MethodGet,
			path:   "/missing",
			mock: shortenerMock{
				expandErr: service.ErrTokenNotFound,
			},
			wantStatus: http.StatusNotFound,
			wantCT:     "text/plain; charset=utf-8",
			wantBody:   "No such URL\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(&tt.mock, &tt.healthCheckMock)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantCT, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.wantLocation, res.Header.Get("Location"))
			assert.Equal(t, tt.wantBody, string(b))
		})
	}
}
