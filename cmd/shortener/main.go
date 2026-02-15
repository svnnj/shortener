package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/handler"
	"github.com/svnnj/shortener/internal/repository"
	"github.com/svnnj/shortener/internal/service"
)

func run() error {
	cfg := config.Get()

	kvStorage, err := repository.NewKVRepository(cfg.FileStoragePath)
	if err != nil {
		slog.Error(err.Error())
	}
	tokenGen := service.NewB64TokenGen(9)
	shortener := service.NewShortener(kvStorage, tokenGen, cfg)
	handler := handler.NewRouter(shortener)

	slog.Info("Starting the server...")
	return http.ListenAndServe(cfg.ServerAddress, handler)
}

var (
	DefaultLogger func(next http.Handler) http.Handler
)

func main() {
	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelDebug},
	))
	slog.SetDefault(logger)

	err := run()
	if err != nil {
		slog.Error(err.Error())
	}
}
