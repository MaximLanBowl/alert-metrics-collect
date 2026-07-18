package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
)

const baseURL = "http://localhost:8080"

func TestMetricsHandler_SetMetrics(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		status int
	}{
		{
			name:   "valid gauge",
			method: http.MethodPost,
			url:    baseURL + "/update/gauge/Lookups/14345",
			status: http.StatusOK,
		},
		{
			name:   "valid counter",
			method: http.MethodPost,
			url:    baseURL + "/update/counter/PollCount/20",
			status: http.StatusOK,
		},
		{
			name:   "invalid metric type",
			method: http.MethodPost,
			url:    baseURL + "/update/invalid/PollCount/20",
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid metric path",
			method: http.MethodPost,
			url:    baseURL + "/update/gauge/Lookups",
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid metric name",
			method: http.MethodPost,
			url:    baseURL + "/update/gauge/",
			status: http.StatusNotFound,
		},
		{
			name:   "invalid method",
			method: http.MethodGet,
			url:    baseURL + "/update/gauge/Lookups/14345",
			status: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			h := NewMetricsHandler(storage)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			h.SetMetrics(w, req)

			if w.Code != tt.status {
				t.Errorf("wrong status code: got %v want %v", w.Code, tt.status)
			}
		})
	}
}
