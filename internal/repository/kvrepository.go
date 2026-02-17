package repository

import (
	"fmt"
	"log/slog"
)

type KVRepository struct {
	mem      *KVStorage
	appender *Appender
}

func NewKVRepository(path string) (*KVRepository, error) {
	loader, err := NewLoader(path)
	if err != nil {
		slog.Error(fmt.Sprintf("loader init: %s", err.Error()))
		return nil, err
	}
	defer loader.file.Close()

	persisted, err := loader.Load()
	if err != nil {
		slog.Error(fmt.Sprintf("load persisted data: %s", err.Error()))
		return nil, err
	}

	mem := NewKVStorage()
	for k, v := range persisted {
		mem.Set(k, v)
	}

	app, err := NewAppender(path)
	if err != nil {
		slog.Error(fmt.Sprintf("appender init: %s", err.Error()))
		return nil, err
	}

	return &KVRepository{
		mem:      mem,
		appender: app,
	}, nil
}

func (r *KVRepository) Get(key string) (string, bool) {
	return r.mem.Get(key)
}

func (r *KVRepository) Set(key, val string) {
	r.mem.Set(key, val)

	rec := &kvRecord{Key: key, Val: val}
	if err := r.appender.Write(rec); err != nil {
		slog.Error(fmt.Sprintf("persist error: %s", err.Error()))
	}
}

func (r *KVRepository) Close() error {
	if r.appender != nil && r.appender.file != nil {
		return r.appender.file.Close()
	}
	return nil
}
