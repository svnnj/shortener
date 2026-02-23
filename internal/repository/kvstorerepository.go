package repository

import (
	"context"
	"database/sql"

	"github.com/svnnj/shortener/internal/config"
)

type KVStoreRepository interface {
	Set(ctx context.Context, key string, val string) error
	Get(ctx context.Context, key string) (string, error)
}

func NewKVStore(cfg config.Config, sqlDB *sql.DB) (KVStoreRepository, error) {
	var urlStore KVStoreRepository
	var err error
	if cfg.DatabaseDSN != "" {
		urlStore = NewKVStoreDB(sqlDB)
	} else if cfg.FileStoragePath != "" {
		urlStore, err = NewKVStorePersistent(cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}
	} else {
		urlStore = NewKVStoreMem()
	}
	return urlStore, nil
}
