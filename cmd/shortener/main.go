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

	db, err := db.NewPostgresDB(cfg.DatabaseDSN)
	if err != nil {
		slog.Error("db init: " + err.Error())
	}
	if db != nil {
		defer db.Close()
	}

	urlStore, err := repository.NewURLStore(cfg, db)
	if err != nil {
		return fmt.Errorf("store init: %w", err)
	}

	tokenGen := service.NewB64TokenGen(9)
	shortener := service.NewShortener(urlStore, tokenGen, cfg)

	healthChecker := service.NewHealthChecker(db)
	handler := handler.NewRouter(shortener, healthChecker)

	slog.Info(fmt.Sprintf(`starting the server on %s`, cfg.ServerAddress))
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
