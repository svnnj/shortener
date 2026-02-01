package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func Get() Config {
	var cfg Config

	cfg.ServerAddress = os.Getenv("SERVER_ADDRESS")
	cfg.BaseURL = os.Getenv("BASE_URL")

	serverAddressFlag := flag.String("a", "localhost:8080", "main server host")
	baseURLFlag := flag.String("b", "http://localhost:8080", "shortened URL server host")
	flag.Parse()

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = *serverAddressFlag
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = *baseURLFlag
	}

	return cfg
}
