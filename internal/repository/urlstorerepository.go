package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/svnnj/shortener/internal/config"
)

var (
	ErrNotFound     = errors.New("no such entity")
	ErrIncorrectKey = errors.New("empty/incorrect key")
)

type URLStoreRepository interface {
	Set(ctx context.Context, rec URLRec) error
	SetBatch(ctx context.Context, urlRecs []URLRec) error
	Get(ctx context.Context, token string) (string, error)
}

type URLRec struct {
	Token       string
	OriginalURL string
}

func NewURLStore(cfg config.Config, sqlDB *sql.DB) (URLStoreRepository, error) {
	var urlStore URLStoreRepository
	var err error
	if cfg.DatabaseDSN != "" {
		urlStore = NewURLStoreDB(sqlDB)
	} else if cfg.FileStoragePath != "" {
		urlStore, err = NewURLStorePersistent(cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}
	} else {
		urlStore = NewURLStoreMem()
	}
	return urlStore, nil
}
