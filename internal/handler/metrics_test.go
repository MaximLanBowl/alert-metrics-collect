package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	models "github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const baseURL = "http://localhost:8080"

func newTestProducer(t *testing.T) *repository.Producer {
	t.Helper()

	path := filepath.Join(t.TempDir(), "metrics.json")
	p, err := repository.NewProducer(path)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	return p
}

func newTestHandler(t *testing.T, storage *repository.MemStorage, cfg config.ServerConfig) *MetricsHandler {
	t.Helper()
	return NewMetrics(storage, newTestProducer(t), cfg)
}

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
			cfg := config.ServerConfig{StoreInterval: 300}
			h := newTestHandler(t, storage, cfg)

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
				storage.SetCounter(t.Context(), "PollCount", 10)
			},
		},
		{
			name:   "valid gauge",
			method: http.MethodGet,
			url:    baseURL + "/value/gauge/Lookups",
			status: http.StatusOK,
			setVal: func(storage *repository.MemStorage) {
				storage.SetGauge(t.Context(), "Lookups", 14345)
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

			cfg := config.ServerConfig{StoreInterval: 300}
			h := newTestHandler(t, storage, cfg)

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
			cfg := config.ServerConfig{StoreInterval: 300}
			h := newTestHandler(t, storage, cfg)

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

func TestMetricsHandler_UpdateMetrics(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		status int
		req    models.Metrics
	}{
		{
			name:   "valid update counter",
			method: http.MethodPost,
			url:    baseURL + "/update",
			status: http.StatusOK,
			req: models.Metrics{
				ID:    "1",
				MType: "counter",
				Delta: func(v int64) *int64 {
					return &v
				}(10),
			},
		},
		{
			name:   "valid update gauge",
			method: http.MethodPost,
			url:    baseURL + "/update",
			status: http.StatusOK,
			req: models.Metrics{
				ID:    "2",
				MType: "gauge",
				Value: func(v float64) *float64 {
					return &v
				}(14345.234),
			},
		},
		{
			name:   "invalid method update",
			method: http.MethodGet,
			url:    baseURL + "/update",
			status: http.StatusMethodNotAllowed,
			req: models.Metrics{
				ID:    "3",
				MType: "gauge",
				Value: func(v float64) *float64 {
					return &v
				}(14345),
			},
		},
		{
			name:   "invalid metric type",
			method: http.MethodPost,
			url:    baseURL + "/update",
			status: http.StatusBadRequest,
			req: models.Metrics{
				ID:    "4",
				MType: "invalid",
				Value: func(v float64) *float64 {
					return &v
				}(14345),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			cfg := config.ServerConfig{StoreInterval: 300}
			h := newTestHandler(t, storage, cfg)

			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("failed to marshal request: %v", err)
			}

			log.Info().
				Str("method", tt.method).
				Str("url", tt.url).
				Str("status", http.StatusText(tt.status)).
				RawJSON("request", body).Msg(
				"Request body",
			)

			req := httptest.NewRequest(tt.method, tt.url, bytes.NewReader(body))
			w := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Post("/update", h.UpdateMetrics)
			r.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("wrong status code: got %v want %v", w.Code, tt.status)
			}
		})
	}
}
