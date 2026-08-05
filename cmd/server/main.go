package main

import (
	"fmt"
	"log"
	"net/http"

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
	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	r := router.New(h)

	zlog.Info().Str("addr", cfg.Address).Msgf("Starting server")
	if err = http.ListenAndServe(cfg.Address, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	zlog.Info().Msg("Server stopped")

	return nil
}
