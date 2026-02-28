package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/svnnj/shortener/internal/model"
	"github.com/svnnj/shortener/internal/service"
)

type MockShortenerService struct{ mock.Mock }

func (m *MockShortenerService) Shorten(ctx context.Context, url string) (string, error) {
	args := m.Called(ctx, url)
	return args.String(0), args.Error(1)
}
func (m *MockShortenerService) ShortenBatch(ctx context.Context, req []model.ShortenBatchReq) ([]model.ShortenBatchRes, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.ShortenBatchRes), args.Error(1)
}
func (m *MockShortenerService) Expand(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

type MockHealthCheckerService struct{ mock.Mock }

func (m *MockHealthCheckerService) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type HandlersSuite struct {
	suite.Suite
	shortenerMock *MockShortenerService
	healthMock    *MockHealthCheckerService
	router        http.Handler
}

func (s *HandlersSuite) SetupTest() {
	s.shortenerMock = new(MockShortenerService)
	s.healthMock = new(MockHealthCheckerService)
	s.router = NewRouter(s.shortenerMock, s.healthMock)
}

func (s *HandlersSuite) request(method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	return rr
}

func (s *HandlersSuite) assertStatus(rr *httptest.ResponseRecorder, expected int) {
	s.Equal(expected, rr.Code, "wrong HTTP status")
}
func (s *HandlersSuite) assertBody(rr *httptest.ResponseRecorder, expected string) {
	s.Equal(expected, strings.TrimSpace(rr.Body.String()), "wrong body")
}
func (s *HandlersSuite) assertHeader(rr *httptest.ResponseRecorder, header, expected string) {
	s.Equal(expected, strings.TrimSpace(rr.Result().Header.Get(header)), "wrong header")
}
func (s *HandlersSuite) resetMocks() {
	s.shortenerMock.ExpectedCalls = nil
	s.shortenerMock.Calls = nil
	s.healthMock.ExpectedCalls = nil
	s.healthMock.Calls = nil
}

func (s *HandlersSuite) TestShortenHandler() {
	tests := []struct {
		name            string
		method          string
		body            string
		headers         map[string]string
		mockReturn      string
		mockError       error
		expectedStatus  int
		expectedBody    string
		expectedHeaders map[string]string
	}{
		{
			name:            "success",
			method:          "POST",
			body:            "https://example.com",
			headers:         nil,
			mockReturn:      "http://short.com/abc123",
			mockError:       nil,
			expectedStatus:  http.StatusCreated,
			expectedBody:    "http://short.com/abc123",
			expectedHeaders: map[string]string{"Content-Type": "text/plain"},
		},
		{
			name:            "invalid url",
			method:          "POST",
			body:            "not a url",
			headers:         nil,
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "Incorrect URL",
			expectedHeaders: nil,
		},
		{
			name:            "service error",
			method:          "POST",
			body:            "https://example.com",
			headers:         nil,
			mockReturn:      "",
			mockError:       errors.New("boom"),
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "Internal error",
			expectedHeaders: nil,
		},
		{
			name:            "body too long",
			method:          "POST",
			body:            "https://example.com" + strings.Repeat("a", 1024),
			headers:         nil,
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "Invalid request body",
			expectedHeaders: nil,
		},
		{
			name:            "method not allowed",
			method:          "PUT",
			body:            "",
			headers:         nil,
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedBody:    "",
			expectedHeaders: nil,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.resetMocks()
			if tt.mockReturn != "" || tt.mockError != nil {
				s.shortenerMock.On("Shorten", mock.Anything, tt.body).Return(tt.mockReturn, tt.mockError)
			}
			rr := s.request(tt.method, "/", []byte(tt.body), tt.headers)
			s.assertStatus(rr, tt.expectedStatus)
			if tt.expectedBody != "" {
				s.assertBody(rr, tt.expectedBody)
			}
			for k, v := range tt.expectedHeaders {
				s.assertHeader(rr, k, v)
			}
			s.shortenerMock.AssertExpectations(s.T())
		})
	}
}

func (s *HandlersSuite) TestExpandHandler() {
	tests := []struct {
		name            string
		token           string
		mockReturn      string
		mockError       error
		expectedStatus  int
		expectedHeaders map[string]string
		expectedBody    string
	}{
		{
			name:            "success",
			token:           "abc123",
			mockReturn:      "https://example.com",
			mockError:       nil,
			expectedStatus:  http.StatusTemporaryRedirect,
			expectedHeaders: map[string]string{"Location": "https://example.com", "Content-Type": "text/plain"},
			expectedBody:    "",
		},
		{
			name:            "not found",
			token:           "abc123",
			mockReturn:      "",
			mockError:       service.ErrTokenNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedHeaders: nil,
			expectedBody:    "",
		},
		{
			name:            "internal error",
			token:           "abc123",
			mockReturn:      "",
			mockError:       errors.New("boom"),
			expectedStatus:  http.StatusInternalServerError,
			expectedHeaders: nil,
			expectedBody:    "",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.resetMocks()
			s.shortenerMock.On("Expand", mock.Anything, tt.token).Return(tt.mockReturn, tt.mockError)
			rr := s.request("GET", "/"+tt.token, nil, nil)
			s.assertStatus(rr, tt.expectedStatus)
			for k, v := range tt.expectedHeaders {
				s.assertHeader(rr, k, v)
			}
			if tt.expectedBody != "" {
				s.assertBody(rr, tt.expectedBody)
			}
			s.shortenerMock.AssertExpectations(s.T())
		})
	}
}

func (s *HandlersSuite) TestPingHandler() {
	tests := []struct {
		name            string
		mockError       error
		expectedStatus  int
		expectedBody    string
		expectedHeaders map[string]string
	}{
		{
			name:            "success",
			mockError:       nil,
			expectedStatus:  http.StatusOK,
			expectedBody:    "",
			expectedHeaders: nil,
		},
		{
			name:            "db not available",
			mockError:       service.ErrNilConnection,
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "Database is not available",
			expectedHeaders: nil,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.resetMocks()
			s.healthMock.On("Ping", mock.Anything).Return(tt.mockError)

			rr := s.request("GET", "/ping", nil, nil)
			s.assertStatus(rr, tt.expectedStatus)
			if tt.expectedBody != "" {
				s.assertBody(rr, tt.expectedBody)
			}
			for k, v := range tt.expectedHeaders {
				s.assertHeader(rr, k, v)
			}
			s.healthMock.AssertExpectations(s.T())
		})
	}
}

func (s *HandlersSuite) TestShortenJSONHandler() {
	tests := []struct {
		name            string
		method          string
		body            string
		headers         map[string]string
		mockURL         string
		mockReturn      string
		mockError       error
		expectedStatus  int
		expectedBody    string
		expectedHeaders map[string]string
	}{
		{
			name:            "success",
			method:          "POST",
			body:            `{"url":"https://example.com"}`,
			headers:         nil,
			mockURL:         "https://example.com",
			mockReturn:      "http://short.com/abc123",
			mockError:       nil,
			expectedStatus:  http.StatusCreated,
			expectedBody:    `{"result":"http://short.com/abc123"}`,
			expectedHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:            "invalid url",
			method:          "POST",
			body:            `{"url":"not a url"}`,
			headers:         nil,
			mockURL:         "not a url",
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "Incorrect URL",
			expectedHeaders: nil,
		},
		{
			name:            "service error",
			method:          "POST",
			body:            `{"url":"https://example.com"}`,
			headers:         nil,
			mockURL:         "https://example.com",
			mockReturn:      "",
			mockError:       errors.New("boom"),
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "Internal error",
			expectedHeaders: nil,
		},
		{
			name:            "body too long",
			method:          "POST",
			body:            `{"url":"https://example.com` + strings.Repeat("a", 1024) + `"}`,
			headers:         nil,
			mockURL:         "",
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "Invalid request body",
			expectedHeaders: nil,
		},
		{
			name:            "method not allowed",
			method:          "PUT",
			body:            `{"url":"https://example.com"}`,
			headers:         nil,
			mockURL:         "",
			mockReturn:      "",
			mockError:       nil,
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedBody:    "",
			expectedHeaders: nil,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.resetMocks()
			if tt.mockReturn != "" || tt.mockError != nil {
				s.shortenerMock.On("Shorten", mock.Anything, tt.mockURL).Return(tt.mockReturn, tt.mockError)
			}
			rr := s.request(tt.method, "/api/shorten", []byte(tt.body), tt.headers)
			s.assertStatus(rr, tt.expectedStatus)
			if tt.expectedBody != "" {
				s.assertBody(rr, tt.expectedBody)
			}
			for k, v := range tt.expectedHeaders {
				s.assertHeader(rr, k, v)
			}
			s.shortenerMock.AssertExpectations(s.T())
		})
	}
}

func (s *HandlersSuite) TestShortenBatchHandler() {
	type testCase struct {
		name           string
		method         string
		body           interface{}
		mockReq        []model.ShortenBatchReq
		mockRes        []model.ShortenBatchRes
		mockError      error
		expectedStatus int
		expectedBody   string
	}
	successReq := []model.ShortenBatchReq{
		{CorrelationID: "1", OriginalURL: "https://example1.com"},
		{CorrelationID: "A", OriginalURL: "https://exampleA.com"},
	}
	successRes := []model.ShortenBatchRes{
		{CorrelationID: "1", ShortURL: "http://short1.com/abc123"},
		{CorrelationID: "A", ShortURL: "http://shortA.com/abc123"},
	}
	successBody, _ := json.Marshal(successReq)
	successResp, _ := json.Marshal(successRes)

	tests := []testCase{
		{
			name:           "success",
			method:         "POST",
			body:           successBody,
			mockReq:        successReq,
			mockRes:        successRes,
			expectedStatus: http.StatusCreated,
			expectedBody:   string(successResp),
		},
		{
			name:   "invalid url",
			method: "POST",
			body: []model.ShortenBatchReq{
				{CorrelationID: "1", OriginalURL: "https://example1.com"},
				{CorrelationID: "A", OriginalURL: "not a url"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Incorrect URL: not a url",
		},
		{
			name:           "service error",
			method:         "POST",
			body:           successBody,
			mockReq:        successReq,
			mockRes:        []model.ShortenBatchRes{},
			mockError:      errors.New("boom"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "body too long",
			method: "POST",
			body: []model.ShortenBatchReq{
				{CorrelationID: "1", OriginalURL: "https://example1.com" + strings.Repeat("a", 1024)},
				{CorrelationID: "A", OriginalURL: "https://exampleA.com"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty array",
			method:         "POST",
			body:           []model.ShortenBatchReq{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty array",
		},
		{
			name:           "method not allowed",
			method:         "PUT",
			body:           []model.ShortenBatchReq{},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.resetMocks()
			var reqBody []byte
			switch v := tt.body.(type) {
			case []byte:
				reqBody = v
			default:
				reqBody, _ = json.Marshal(v)
			}
			if tt.mockReq != nil {
				s.shortenerMock.On("ShortenBatch", mock.Anything, tt.mockReq).Return(tt.mockRes, tt.mockError)
			}
			rr := s.request(tt.method, "/api/shorten/batch", reqBody, nil)
			s.assertStatus(rr, tt.expectedStatus)
			if tt.expectedBody != "" {
				s.assertBody(rr, tt.expectedBody)
			}
			s.shortenerMock.AssertExpectations(s.T())
		})
	}
}

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(HandlersSuite))
}
