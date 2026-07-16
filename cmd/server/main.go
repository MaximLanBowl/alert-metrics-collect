package main

import (
	"fmt"
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/router"
	"github.com/rs/zerolog/log"
)

const addr = "localhost:8080"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	r := router.New(h)

	log.Info().Str("addr", addr).Msgf("Starting server")
	if err := http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	log.Info().Msg("Server stopped")
	
	return nil
}
