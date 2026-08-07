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

	r.Post("/update/{type}/{name}/{value}", h.SetMetrics)
	r.Get("/value/{type}/{name}", h.GetMetrics)
	r.Get("/", h.GetMetricsList)

	return r
}
