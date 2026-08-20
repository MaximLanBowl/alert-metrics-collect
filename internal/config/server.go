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

func LoadServer(args []string) (ServerConfig, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	addr := fs.String("a", "localhost:8080", "server listen address")
	storeInterval := fs.Int("i", 300, "interval to store data")
	filePath := fs.String("f", "runtime_metrics.json", "file for runtime metrics")
	restore := fs.Bool("r", false, "restore data from file")

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to parse flags: %w", err)
	}

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
