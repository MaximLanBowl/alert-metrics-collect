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

	var ping handler.Pinger
	if cfg.Postgres.DatabaseDSN != "" {
		connPool, err := db.Init(cfg.Postgres)
		if err != nil {
			zlog.Error().Err(err).Msg("db unavailable, /ping will return 500")
		} else {
			defer connPool.Close()
			ping = connPool
		}
	}

	storage := repository.NewMemStorage()

	producer, err := repository.NewProducer(cfg.Server.FileStoragePath)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}

	h := &handler.Handlers{
		Metrics: handler.NewMetrics(storage, producer, cfg.Server),
		Ping:    handler.NewPingDB(ping),
	}
	defer h.Metrics.CloseMetricsFile()

	if cfg.Server.Restore {
		zlog.Info().Msgf("Restoring metrics from file %s", cfg.Server.FileStoragePath)
		consumer, err := repository.NewConsumer(cfg.Server.FileStoragePath)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to create consumer")
			return fmt.Errorf("failed to create consumer: %w", err)
		}
		defer consumer.Close()

		if err = h.Metrics.Restore(consumer); err != nil {
			zlog.Error().Err(err).Msg("Failed to restore metrics")
			return fmt.Errorf("failed to restore metrics: %w", err)
		}
	}

	h.Metrics.StartAutoSave()

	r := router.New(h)

	zlog.Info().Str("addr", cfg.Server.Address).Msgf("Starting server")
	if err = http.ListenAndServe(cfg.Server.Address, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	zlog.Info().Msg("Server stopped")

	return nil
}
