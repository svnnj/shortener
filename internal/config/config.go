package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
}

func Get() Config {
	var cfg Config

	cfg.ServerAddress = os.Getenv("SERVER_ADDRESS")
	cfg.BaseURL = os.Getenv("BASE_URL")
	cfg.FileStoragePath = os.Getenv("FILE_STORAGE_PATH")
	cfg.DatabaseDSN = os.Getenv("DATABASE_DSN")

	serverAddressFlag := flag.String("a", "localhost:8080", "main server host")
	baseURLFlag := flag.String("b", "http://localhost:8080", "shortened URL server host")
	fileStoragePathFlag := flag.String("f", "./kv.txt", "storage path")
	datapaseDSNFlag := flag.String("d", "", "database dsn string")
	flag.Parse()

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = *serverAddressFlag
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = *baseURLFlag
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = *fileStoragePathFlag
	}
	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = *datapaseDSNFlag
	}

	return cfg
}
