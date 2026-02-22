package repository

import (
	"sync"
)

type KVStorageRepository interface {
	Set(key string, val string)
	Get(key string) (string, bool)
}

type KVStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewKVStorage() *KVStorage {

	return &KVStorage{
		mu:   sync.RWMutex{},
		data: map[string]string{},
	}
}

func (s *KVStorage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, exists := s.data[key]
	return val, exists
}

func (s *KVStorage) Set(key string, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}
