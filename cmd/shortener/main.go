package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/config/db"
	"github.com/svnnj/shortener/internal/handler"
	"github.com/svnnj/shortener/internal/repository"
	"github.com/svnnj/shortener/internal/service"
)

func run(ctx context.Context) error {
	cfg := config.Get()

	kvStorage, err := repository.NewKVRepository(cfg.FileStoragePath)
	if err != nil {
		return fmt.Errorf("key value repository init: %w", err)
	}
	defer kvStorage.Close()
	tokenGen := service.NewB64TokenGen(9)
	shortener := service.NewShortener(kvStorage, tokenGen, cfg)

	sqlDB, err := db.NewPostgresDB(cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("db init: %w", err)
	}
	defer sqlDB.Close()

	healthChecker := service.NewHealthChecker(sqlDB)
	handler := handler.NewRouter(shortener, healthChecker)

	slog.Info("starting the server...")
	return http.ListenAndServe(cfg.ServerAddress, handler)
}

var (
	DefaultLogger func(next http.Handler) http.Handler
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelDebug},
	))
	slog.SetDefault(logger)

	err := run(ctx)
	if err != nil {
		slog.Error(err.Error())
	}
}
