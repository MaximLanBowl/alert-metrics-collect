package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	PollInterval   int64  `env:"POLL_INTERVAL"`
	SecretKey      string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func LoadAgent(args []string) (AgentConfig, error) {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	addr := fs.String("a", "localhost:8080", "server listen address")
	reportInterval := fs.Int64("r", 10, "report timeout")
	pollInterval := fs.Int64("p", 2, "poll timeout")
	skey := fs.String("k", "", "agent secret key")
	rlimit := fs.Int("l", 30, "rate limit")

	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse flags: %w", err)
	}

	cfg := AgentConfig{
		Address:        *addr,
		ReportInterval: *reportInterval,
		PollInterval:   *pollInterval,
		SecretKey:      *skey,
		RateLimit:      *rlimit,
	}

	if err := env.Parse(&cfg); err != nil {
		return AgentConfig{}, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
