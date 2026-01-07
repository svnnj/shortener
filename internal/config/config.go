package config

import (
	"flag"
)

type Config struct {
	Host          string
	ShortenedHost string
}

func Get() Config {
	host := flag.String("a", "localhost:8080", "main server host")
	shortenedHost := flag.String("b", "http://localhost:8080", "shortened URL server host")
	flag.Parse()

	return Config{
		Host:          *host,
		ShortenedHost: *shortenedHost,
	}
}
