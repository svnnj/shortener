package repository

import (
	"context"
	"fmt"
)

type urlStorePersistent struct {
	cache  *urlStoreMem
	writer *urlStoreWriter
}

func NewURLStorePersistent(path string) (*urlStorePersistent, error) {
	w, err := NewURLStoreWriter(path)
	if err != nil {
		return nil, fmt.Errorf("create persistent store with path %q: %w", path, err)
	}

	return &urlStorePersistent{
		cache:  NewURLStoreMem(),
		writer: w,
	}, nil
}

func (s *urlStorePersistent) Load(ctx context.Context, path string) error {
	reader, err := NewURLStoreReader(path)
	if err != nil {
		return fmt.Errorf("open reader for loading from %q: %w", path, err)
	}
	defer reader.Close()

	persisted, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read all from %q: %w", path, err)
	}

	for k, v := range persisted {
		s.cache.Set(ctx, URLRec{Token: k, OriginalURL: v})
	}

	return nil
}

func (s *urlStorePersistent) Get(ctx context.Context, token string) (string, error) {
	url, err := s.cache.Get(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get from persistent cache for token %q: %w", token, err)
	}
	return url, nil
}

func (s *urlStorePersistent) Set(ctx context.Context, rec URLRec) error {
	s.cache.Set(ctx, rec)

	if err := s.writer.Put(rec); err != nil {
		return fmt.Errorf("persistent set for token %q: %w", rec.Token, err)
	}

	return nil
}

func (s *urlStorePersistent) SetBatch(ctx context.Context, urlRecs []URLRec) error {
	for _, v := range urlRecs {
		s.cache.Set(ctx, v)
	}

	if err := s.writer.PutBatch(urlRecs); err != nil {
		return fmt.Errorf("persistent batch set for %d records: %w", len(urlRecs), err)
	}

	return nil
}

func (s *urlStorePersistent) Close() error {
	if s.writer != nil {
		return s.writer.Close()
	}
	return nil
}
