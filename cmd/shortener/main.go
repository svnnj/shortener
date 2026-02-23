package main

import (
	"context"
	"database/sql"
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

	var sqlDB *sql.DB
	var err error

	if cfg.DatabaseDSN != "" {
		sqlDB, err = db.NewPostgresDB(cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("db init: %w", err)
		}
		defer sqlDB.Close()
	}

	urlStore, err := repository.NewKVStore(cfg, sqlDB)
	if err != nil {
		return fmt.Errorf("store init: %w", err)
	}

	tokenGen := service.NewB64TokenGen(9)
	shortener := service.NewShortener(urlStore, tokenGen, cfg)

	healthChecker := service.NewHealthChecker(sqlDB)
	handler := handler.NewRouter(shortener, healthChecker)

	slog.Info("starting the server...")
	return http.ListenAndServe(cfg.ServerAddress, handler)
}

var (
	DefaultLogger func(next http.Handler) http.Handler
)

func main() {
	ctx := context.TODO()
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
