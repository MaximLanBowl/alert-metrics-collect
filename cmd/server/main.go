package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/router"
	"github.com/rs/zerolog/log"
)

var flagRunAddr string

func main() {
	parseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	r := router.New(h)

	log.Info().Str("addr", flagRunAddr).Msgf("Starting server")
	if err := http.ListenAndServe(flagRunAddr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	log.Info().Msg("Server stopped")

	return nil
}

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "server listen address")
	flag.Parse()
}
