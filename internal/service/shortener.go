package service

import (
	"errors"
	"fmt"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/repository"
)

type ShortenerService interface {
	Shorten(originalURL string) (string, error)
	Expand(token string) (string, error)
}

type Shortener struct {
	kvStorage repository.KVStorageRepository
	tokenGen  TokenService
	cfg       config.Config
}

func NewShortener(kvStorage repository.KVStorageRepository, tokenGen TokenService, cfg config.Config) *Shortener {
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

func (sh *Shortener) Shorten(originalURL string) (string, error) {
	var (
		token string
		err   error
	)

	for i := 0; true; i++ {
		token = sh.tokenGen.Generate()
		if _, exists := sh.kvStorage.Get(token); !exists {
			break
		}
		if i >= 3 {
			err = ErrTokenCollision
			return "", err
		}
	}

	sh.kvStorage.Set(token, originalURL)
	shortURL := fmt.Sprintf("%s/%s", sh.cfg.BaseURL, token)

	return shortURL, nil
}

func (sh *Shortener) Expand(token string) (string, error) {
	var err error
	originalURL, exists := sh.kvStorage.Get(token)
	if !exists {
		err = ErrTokenNotFound
	}
	return originalURL, err
}
