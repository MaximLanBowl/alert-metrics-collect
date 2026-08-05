package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type ServerConfig struct {
	Address string `env:"ADDRESS"`
}

func LoadServer() (ServerConfig, error) {
	addr := flag.String("a", "localhost:8080", "server listen address")
	flag.Parse()

	cfg := ServerConfig{
		Address: *addr,
	}

	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
