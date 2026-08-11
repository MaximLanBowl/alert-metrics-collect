package router

import (
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/middleware/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(h *handler.MetricsHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(logger.WithLogging)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	// POST
	r.Post("/update/{type}/{name}/{value}", h.SetMetrics)
	r.Post("/update", h.UpdateMetrics)
	r.Post("/value", h.GetMetricsByValue)

	// GET
	r.Get("/value/{type}/{name}", h.GetMetrics)
	r.Get("/", h.GetMetricsList)

	return r
}
