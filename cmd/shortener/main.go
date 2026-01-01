package main

import (
	"log"
	"net/http"

	"github.com/svnnj/shortener/internal/config"
	"github.com/svnnj/shortener/internal/handler"
	"github.com/svnnj/shortener/internal/repository"
	"github.com/svnnj/shortener/internal/service"
)

func run() error {
	cfg := config.Get()

	kvStorage := repository.NewKVStorage()
	tokenGen := service.NewB64TokenGen(9)
	shortener := service.NewShortener(kvStorage, tokenGen, cfg)
	handler := handler.NewRouter(shortener)

	return http.ListenAndServe(cfg.Host, handler)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
