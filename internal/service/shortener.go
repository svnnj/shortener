package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/model"
	"github.com/svnnj/shortener/internal/repository"
)

var (
	ErrTokenCollision = errors.New("token collision happened")
	ErrTokenNotFound  = errors.New("token not found")
)

type (
	ShortenerService interface {
		Shorten(ctx context.Context, originalURL string) (string, error)
		ShortenBatch(ctx context.Context, originalURLs []model.ShortenBatchReq) ([]model.ShortenBatchRes, error)
		Expand(ctx context.Context, token string) (string, error)
	}

	Shortener struct {
		urlStorage repository.URLStoreRepository
		tokenGen   TokenService
		cfg        config.Config
	}
)

func NewShortener(urlStorage repository.URLStoreRepository, tokenGen TokenService, cfg config.Config) *Shortener {
	return &Shortener{
		urlStorage: urlStorage,
		tokenGen:   tokenGen,
		cfg:        cfg,
	}
}

func (sh *Shortener) Shorten(ctx context.Context, originalURL string) (string, error) {
	var (
		token string
		err   error
	)

	for i := 0; true; i++ {
		token = sh.tokenGen.Generate()
		if _, err := sh.urlStorage.Get(ctx, token); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				break
			}
			return "", fmt.Errorf("checking token (%s) for url (%s). %w", token, originalURL, err)
		}
		if i >= 3 {
			err = ErrTokenCollision
			return "", err
		}
	}

	sh.urlStorage.Set(ctx, repository.URLRec{Token: token, OriginalURL: originalURL})
	shortURL := fmt.Sprintf("%s/%s", sh.cfg.BaseURL, token)

	return shortURL, nil
}

func (sh *Shortener) ShortenBatch(ctx context.Context, originalURLs []model.ShortenBatchReq) ([]model.ShortenBatchRes, error) {
	var batchReqs []model.ShortenBatchRes
	var urlRecs []repository.URLRec
	for _, v := range originalURLs {
		var (
			token string
		)
		for i := 0; true; i++ {
			token = sh.tokenGen.Generate()
			if _, err := sh.urlStorage.Get(ctx, token); err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					break
				}
				return nil, fmt.Errorf("checking token (%s) for correlation_id (%s). %w", token, v.CorrelationID, err)
			}
			if i >= 3 {
				return []model.ShortenBatchRes{}, fmt.Errorf("generating token for correlation_id (%s): %w", v.CorrelationID, ErrTokenCollision)
			}
		}
		batchReqs = append(batchReqs, model.ShortenBatchRes{CorrelationID: v.CorrelationID, ShortURL: fmt.Sprintf("%s/%s", sh.cfg.BaseURL, token)})
		urlRecs = append(urlRecs, repository.URLRec{Token: token, OriginalURL: v.OriginalURL})
	}

	err := sh.urlStorage.SetBatch(ctx, urlRecs)
	if err != nil {
		return []model.ShortenBatchRes{}, fmt.Errorf("persisting batch: %w", err)
	}

	return batchReqs, nil
}

func (sh *Shortener) Expand(ctx context.Context, token string) (string, error) {
	var err error
	originalURL, err := sh.urlStorage.Get(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", fmt.Errorf("token (%s) not found. %w", token, ErrTokenNotFound)
		}
		return "", fmt.Errorf("expanding token (%s). %w", token, err)
	}
	return originalURL, err
}
