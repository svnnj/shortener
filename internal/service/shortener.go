package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/repository"
)

type ShortenerService interface {
	Shorten(ctx context.Context, originalURL string) (string, error)
	Expand(ctx context.Context, token string) (string, error)
}

type Shortener struct {
	kvStorage repository.KVStoreRepository
	tokenGen  TokenService
	cfg       config.Config
}

func NewShortener(kvStorage repository.KVStoreRepository, tokenGen TokenService, cfg config.Config) *Shortener {
	return &Shortener{
		kvStorage: kvStorage,
		tokenGen:  tokenGen,
		cfg:       cfg,
	}
}

var (
	ErrTokenCollision = errors.New("token collision happened")
	ErrTokenNotFound  = errors.New("token not found")
)

func (sh *Shortener) Shorten(ctx context.Context, originalURL string) (string, error) {
	var (
		token string
		err   error
	)

	for i := 0; true; i++ {
		token = sh.tokenGen.Generate()
		if _, err := sh.kvStorage.Get(ctx, token); err != nil {
			break
		}
		if i >= 3 {
			err = ErrTokenCollision
			return "", err
		}
	}

	sh.kvStorage.Set(ctx, token, originalURL)
	shortURL := fmt.Sprintf("%s/%s", sh.cfg.BaseURL, token)

	return shortURL, nil
}

func (sh *Shortener) Expand(ctx context.Context, token string) (string, error) {
	var err error
	originalURL, err := sh.kvStorage.Get(ctx, token)
	if err != nil {
		err = ErrTokenNotFound
	}
	return originalURL, err
}
