package repository

import (
	"context"
	"fmt"
	"log/slog"
)

type kvStorePersistent struct {
	cache  *kvStoreMem
	writer *kvStoreWriter
	reader *kvStoreReader
}

func NewKVStorePersistent(path string) (*kvStorePersistent, error) {
	reader, err := NewKVStoreReader(path)
	if err != nil {
		slog.Error(fmt.Sprintf("reader init: %s", err))
		return nil, err
	}
	defer reader.Close()

	w, err := NewKVStoreWriter(path)
	if err != nil {
		slog.Error(fmt.Sprintf("writer init: %s", err))
		return nil, err
	}

	return &kvStorePersistent{
		cache:  NewKVStoreMem(),
		writer: w,
	}, nil
}

func (s *kvStorePersistent) Load(ctx context.Context) error {
	persisted, err := s.reader.ReadAll()
	if err != nil {
		slog.Error(fmt.Sprintf("load persisted data: %s", err))
		return err
	}

	for k, v := range persisted {
		s.cache.Set(ctx, k, v)
	}

	return nil
}

func (s *kvStorePersistent) Get(ctx context.Context, key string) (string, error) {
	val, err := s.cache.Get(ctx, key)
	if err != nil {
		return val, err
	}
	return val, nil
}

func (s *kvStorePersistent) Set(ctx context.Context, key string, val string) error {
	s.cache.Set(ctx, key, val)

	if err := s.writer.Put(&kvEntry{Key: key, Val: val}); err != nil {
		slog.Error(fmt.Sprintf("persist error: %s", err))
		return err
	}

	return nil
}

func (s *kvStorePersistent) Close() error {
	if s.writer != nil {
		return s.writer.Close()
	}
	return nil
}
