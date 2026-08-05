package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

func Load() (Config, error) {
	addr := flag.String("a", "localhost:8080", "server listen address")
	reportInterval := flag.Int("r", 10, "report timeout")
	pollInterval := flag.Int("p", 2, "poll timeout")
	flag.Parse()

	cfg := Config{
		Address:        *addr,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
	}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
