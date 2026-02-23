package repository

import (
	"context"
	"fmt"
	"sync"
)

type kvStoreMem struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewKVStoreMem() *kvStoreMem {
	return &kvStoreMem{
		mu:    sync.RWMutex{},
		store: make(map[string]string),
	}
}

func (s *kvStoreMem) Get(ctx context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.store[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return val, nil
}

func (s *kvStoreMem) Set(ctx context.Context, key string, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = val
	return nil
}
