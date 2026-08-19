package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/router"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadServer(os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	storage := repository.NewMemStorage()

	producer, err := repository.NewProducer(cfg.FileStoragePath)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}

	h := handler.NewMetrics(storage, producer, cfg)
	defer h.CloseMetricsFile()

	if cfg.Restore {
		zlog.Info().Msgf("Restoring metrics from file %s", cfg.FileStoragePath)
		consumer, err := repository.NewConsumer(cfg.FileStoragePath)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to create consumer")
			return fmt.Errorf("failed to create consumer: %w", err)
		}
		defer consumer.Close()

		if err = h.Restore(consumer); err != nil {
			zlog.Error().Err(err).Msg("Failed to restore metrics")
			return fmt.Errorf("failed to restore metrics: %w", err)
		}
	}

	h.StartAutoSave()

	r := router.New(h)

	zlog.Info().Str("addr", cfg.Address).Msgf("Starting server")
	if err = http.ListenAndServe(cfg.Address, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	zlog.Info().Msg("Server stopped")

	return nil
}
