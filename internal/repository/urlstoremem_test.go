package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type URLStoreMemTestSuite struct {
	suite.Suite
	store *urlStoreMem
	ctx   context.Context
}

func TestURLStoreMemSuite(t *testing.T) {
	suite.Run(t, new(URLStoreMemTestSuite))
}

func (s *URLStoreMemTestSuite) SetupTest() {
	s.store = NewURLStoreMem()
	s.ctx = context.Background()
}

func (s *URLStoreMemTestSuite) TearDownTest() {
	s.store = nil
}

func (s *URLStoreMemTestSuite) TestGet() {
	testRecords := []URLRec{
		{Token: "abc123", OriginalURL: "https://example.com"},
		{Token: "def456", OriginalURL: "https://google.com"},
	}

	for _, rec := range testRecords {
		err := s.store.Set(s.ctx, rec)
		s.Require().NoError(err)
	}

	tests := []struct {
		name        string
		token       string
		expectedURL string
		expectedErr error
	}{
		{
			name:        "successful",
			token:       "abc123",
			expectedURL: "https://example.com",
			expectedErr: nil,
		},
		{
			name:        "not found",
			token:       "nonexistent",
			expectedURL: "",
			expectedErr: ErrNotFound,
		},
		{
			name:        "empty token",
			token:       "",
			expectedURL: "",
			expectedErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			url, err := s.store.Get(s.ctx, tt.token)

			if tt.expectedErr != nil {
				s.ErrorIs(err, tt.expectedErr)
				s.Empty(url)
			} else {
				s.NoError(err)
				s.Equal(tt.expectedURL, url)
			}
		})
	}
}

func (s *URLStoreMemTestSuite) TestSet() {
	tests := []struct {
		name          string
		record        URLRec
		expectError   bool
		expectedError error
	}{
		{
			name: "sucessful",
			record: URLRec{
				Token:       "token1",
				OriginalURL: "https://example1.com",
			},
			expectError: false,
		},
		{
			name: "successful override",
			record: URLRec{
				Token:       "token1",
				OriginalURL: "https://updated.com",
			},
			expectError: false,
		},
		{
			name: "empty token",
			record: URLRec{
				Token:       "",
				OriginalURL: "https://example2.com",
			},
			expectError:   true,
			expectedError: ErrIncorrectKey,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.store.Set(s.ctx, tt.record)

			if tt.expectError {
				s.ErrorIs(err, tt.expectedError)
			} else {
				s.NoError(err)

				url, err := s.store.Get(s.ctx, tt.record.Token)
				s.NoError(err)
				s.Equal(tt.record.OriginalURL, url)
			}
		})
	}
}

func (s *URLStoreMemTestSuite) TestSetBatch() {
	tests := []struct {
		name        string
		records     []URLRec
		expectError bool
		checkRecord URLRec
	}{
		{
			name: "successful",
			records: []URLRec{
				{Token: "batch1", OriginalURL: "https://batch1.com"},
				{Token: "batch2", OriginalURL: "https://batch2.com"},
				{Token: "batch3", OriginalURL: "https://batch3.com"},
			},
			expectError: false,
			checkRecord: URLRec{Token: "batch2", OriginalURL: "https://batch2.com"},
		},
		{
			name: "successful one rec",
			records: []URLRec{
				{Token: "single", OriginalURL: "https://single.com"},
			},
			expectError: false,
			checkRecord: URLRec{Token: "single", OriginalURL: "https://single.com"},
		},
		{
			name:        "succesful empty batch",
			records:     []URLRec{},
			expectError: false,
			checkRecord: URLRec{Token: "nonexistent", OriginalURL: ""},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.store.SetBatch(s.ctx, tt.records)

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)

				for _, rec := range tt.records {
					url, err := s.store.Get(s.ctx, rec.Token)
					s.NoError(err)
					s.Equal(rec.OriginalURL, url)
				}
			}
		})
	}
}

func (s *URLStoreMemTestSuite) TestConcurrentAccess() {
	records := []URLRec{
		{Token: "conc1", OriginalURL: "https://conc1.com"},
		{Token: "conc2", OriginalURL: "https://conc2.com"},
		{Token: "conc3", OriginalURL: "https://conc3.com"},
	}

	s.Run("Concurrent writes", func() {
		done := make(chan bool)
		for _, rec := range records {
			go func(r URLRec) {
				err := s.store.Set(s.ctx, r)
				s.NoError(err)
				done <- true
			}(rec)
		}

		for i := 0; i < len(records); i++ {
			<-done
		}
	})

	s.Run("Concurrent reads", func() {
		done := make(chan bool)
		for _, rec := range records {
			go func(r URLRec) {
				url, err := s.store.Get(s.ctx, r.Token)
				s.NoError(err)
				s.Equal(r.OriginalURL, url)
				done <- true
			}(rec)
		}

		for i := 0; i < len(records); i++ {
			<-done
		}
	})
}

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(URLStoreMemTestSuite))
}
