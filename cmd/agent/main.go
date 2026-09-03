package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	collector := agent.NewMemCollect(cfg)
	collector.Run(ctx)
	collector.Close()

	return nil
}
