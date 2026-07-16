package router

import (
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/handler"
)

func New(h *handler.MetricsHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", h.SetMetrics)
	
	return mux
}
