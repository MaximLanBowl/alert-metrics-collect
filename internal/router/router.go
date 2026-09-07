package router

import (
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	mdv "github.com/MaximLanBowl/alert-metrics-collect/internal/middleware"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/middleware/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(h *handler.Handlers, key string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	// Custom Middlewares
	r.Use(mdv.Compressor)
	r.Use(logger.WithLogging)

	r.Post("/update/{type}/{name}/{value}", h.Metrics.SetMetrics)
	r.Get("/ping", h.Ping.PingDatabase)

	// Subscribed HS256 GROUP
	r.Group(func(r chi.Router) {
		r.Use(mdv.HashSH256(key))
		r.Post("/updates", h.Metrics.UpdateMetricsBatch)
		r.Post("/update", h.Metrics.UpdateMetrics)
		r.Post("/value", h.Metrics.GetMetricsByValue)
		r.Get("/value/{type}/{name}", h.Metrics.GetMetrics)
		r.Get("/", h.Metrics.GetMetricsList)
	})

	return r
}
