package main

import (
	"fmt"
	"log"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadAgent()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	collector := agent.NewMemCollect(cfg)
	collector.Run()

	return nil
}
