package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func LoadAgent(args []string) (AgentConfig, error) {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	addr := fs.String("a", "localhost:8080", "server listen address")
	reportInterval := fs.Int("r", 10, "report timeout")
	pollInterval := fs.Int("p", 2, "poll timeout")

	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse flags: %w", err)
	}

	cfg := AgentConfig{
		Address:        *addr,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
	}

	if err := env.Parse(&cfg); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
