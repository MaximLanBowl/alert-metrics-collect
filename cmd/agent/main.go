package main

import (
	"fmt"
	"log"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
)

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	collector := agent.NewMemCollect(cfg)
	collector.Run()

	return nil
}
