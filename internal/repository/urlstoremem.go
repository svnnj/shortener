package repository

import (
	"context"
	"fmt"
	"sync"
)

type urlStoreMem struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewURLStoreMem() *urlStoreMem {
	return &urlStoreMem{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}
}

func (s *urlStoreMem) Get(ctx context.Context, token string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.store[token]
	if !ok {
		return "", fmt.Errorf("getting url for token: %s. %w", token, ErrNotFound)
	}
	return url, nil
}

func (s *urlStoreMem) Set(ctx context.Context, rec URLRec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[rec.Token] = rec.OriginalURL
	return nil
}

func (s *urlStoreMem) SetBatch(ctx context.Context, urlRecs []URLRec) error {
	for _, v := range urlRecs {
		err := s.Set(ctx, v)
		if err != nil {
			return fmt.Errorf("saving batch: %w", err)
		}
	}
	return nil
}
