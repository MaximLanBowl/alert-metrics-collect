package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/db"
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
	flags, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	cfg, err := config.Load(*flags)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var (
		storage  handler.MetricsStorage
		producer handler.MetricsProducer
		ping     handler.Pinger
	)

	switch {
	case cfg.Postgres.DatabaseDSN != "":
		connPool, err := db.Init(cfg.Postgres)
		if err != nil {
			return fmt.Errorf("failed to init postgres: %w", err)
		}
		defer connPool.Close()

		storage = repository.NewDBStorage(connPool)
		ping = connPool

	case cfg.Server.FileStoragePath != "":
		producer, err = repository.NewProducer(
			cfg.Server.FileStoragePath,
		)
		if err != nil {
			return fmt.Errorf("failed to create producer: %w", err)
		}

		storage = repository.NewMemStorage()

	default:
		storage = repository.NewMemStorage()
	}

	h := &handler.Handlers{
		Metrics: handler.NewMetrics(
			storage,
			producer,
			cfg.Server,
		),
		Ping: handler.NewPingDB(ping),
	}

	defer h.Metrics.CloseMetricsFile()

	if producer != nil && cfg.Server.Restore {
		zlog.Info().
			Msgf(
				"Restoring metrics from file %s",
				cfg.Server.FileStoragePath,
			)

		consumer, err := repository.NewConsumer(
			cfg.Server.FileStoragePath,
		)
		if err != nil {
			return fmt.Errorf("failed to create consumer: %w", err)
		}
		defer consumer.Close()

		if err := h.Metrics.Restore(consumer); err != nil {
			return fmt.Errorf("failed to restore metrics: %w", err)
		}
	}

	h.Metrics.StartAutoSave()

	r := router.New(h)

	zlog.Info().
		Str("addr", cfg.Server.Address).
		Msg("Starting server")

	if err := http.ListenAndServe(cfg.Server.Address, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
