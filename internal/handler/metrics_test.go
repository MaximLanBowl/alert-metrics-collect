package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/go-chi/chi/v5"
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
			status: http.StatusNotFound,
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

			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", h.SetMetrics)
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("wrong status code: got %v want %v", w.Code, tt.status)
			}
		})
	}
}

func TestMetricsHandler_GetMetrics(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		status int
		setVal func(storage *repository.MemStorage)
	}{
		{
			name:   "valid counter",
			method: http.MethodGet,
			url:    baseURL + "/value/counter/PollCount",
			status: http.StatusOK,
			setVal: func(storage *repository.MemStorage) {
				storage.AddCounter("PollCount", 10)
			},
		},
		{
			name:   "valid gauge",
			method: http.MethodGet,
			url:    baseURL + "/value/gauge/Lookups",
			status: http.StatusOK,
			setVal: func(storage *repository.MemStorage) {
				storage.SetGauge("Lookups", 14345)
			},
		},
		{
			name:   "empty metric",
			method: http.MethodGet,
			url:    baseURL + "/value/gauge/None",
			status: http.StatusNotFound,
			setVal: func(storage *repository.MemStorage) {},
		},
		{
			name:   "invalid metric type",
			method: http.MethodGet,
			url:    baseURL + "/value/invalid/Alloc",
			status: http.StatusBadRequest,
			setVal: func(storage *repository.MemStorage) {},
		},
		{
			name:   "invalid metric path",
			method: http.MethodGet,
			url:    baseURL + "/value/gauge",
			status: http.StatusNotFound,
			setVal: func(storage *repository.MemStorage) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			tt.setVal(storage)

			h := NewMetricsHandler(storage)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", h.GetMetrics)
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("wrong status code: got %v want %v", w.Code, tt.status)
			}
		})
	}
}

func TestMetricsHandler_GetMetricsList(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		status int
	}{
		{
			name:   "valid list metrics",
			method: http.MethodGet,
			url:    baseURL + "/",
			status: http.StatusOK,
		},
		{
			name:   "invalid method",
			method: http.MethodPost,
			url:    baseURL + "/",
			status: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			h := NewMetricsHandler(storage)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Get("/", h.GetMetricsList)
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("wrong status code: got %v want %v", w.Code, tt.status)
			}
		})
	}
}
