package router

import (
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	gzipmdv "github.com/MaximLanBowl/alert-metrics-collect/internal/middleware"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/middleware/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(h *handler.Handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	
	// Custom
	r.Use(gzipmdv.Compressor)
	r.Use(logger.WithLogging)

	// POST
	r.Post("/update/{type}/{name}/{value}", h.Metrics.SetMetrics)
	r.Post("/updates", h.Metrics.UpdateMetricsBatch)
	r.Post("/update", h.Metrics.UpdateMetrics)
	r.Post("/value", h.Metrics.GetMetricsByValue)

	// GET
	r.Get("/ping", h.Ping.PingDatabase)
	r.Get("/value/{type}/{name}", h.Metrics.GetMetrics)
	r.Get("/", h.Metrics.GetMetricsList)

	return r
}
