package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func LoadServer() (ServerConfig, error) {
	addr := flag.String("a", "localhost:8080", "server listen address")
	storeInterval := flag.Int("i", 300, "interval to store data")
	filePath := flag.String("f", "runtime_metrics.json", "file for runtime metrics")
	restore := flag.Bool("r", false, "restore data from file")

	flag.Parse()

	cfg := ServerConfig{
		Address:         *addr,
		StoreInterval:   *storeInterval,
		FileStoragePath: *filePath,
		Restore:         *restore,
	}

	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
