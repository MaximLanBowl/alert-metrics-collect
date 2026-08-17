package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/agent"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadAgent(os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	flag.Parse()

	collector := agent.NewMemCollect(cfg)
	collector.Run()

	return nil
}
