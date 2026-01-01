package config

type Config struct {
	Host     string
	Protocol string
}

func Get() Config {
	return Config{
		Host:     "localhost:8080",
		Protocol: "http",
	}
}
