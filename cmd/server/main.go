package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/router"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	var addr string
	parseFlags(&addr)

	if err := run(addr); err != nil {
		log.Fatal(err)
	}
}

func run(addr string) error {
	storage := repository.NewMemStorage()
	h := handler.NewMetricsHandler(storage)
	r := router.New(h)

	zlog.Info().Str("addr", addr).Msgf("Starting server")
	if err := http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	zlog.Info().Msg("Server stopped")

	return nil
}

func parseFlags(addr *string) {
	flag.StringVar(addr, "a", "localhost:8080", "server listen address")
	flag.Parse()
}
