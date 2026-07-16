package handler

import (
	"net/http"
	"strconv"
	"strings"

	models "github.com/MaximLanBowl/alert-metrics-collect/internal/model"
)

type MetricsStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
}

type MetricsHandler struct {
	metricHandler MetricsStorage
}

func NewMetricsHandler(metricHandler MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		metricHandler: metricHandler,
	}
}

func (m *MetricsHandler) SetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Metric name is empty", http.StatusNotFound)
		return
	}
	if len(parts) != 4 || parts[3] == "" {
		http.Error(w, "Metric path is not valid", http.StatusBadRequest)
		return
	}

	mtype := parts[1]
	mname := parts[2]
	mvalue := parts[3]

	switch mtype {
	case models.Gauge:
		mvalue, err := strconv.ParseFloat(mvalue, 64)
		if err != nil {
			http.Error(w, "Metric gauge value is not a number", http.StatusBadRequest)
			return
		}
		m.metricHandler.SetGauge(mname, mvalue)
	case models.Counter:
		mvalue, err := strconv.ParseInt(mvalue, 10, 64)
		if err != nil {
			http.Error(w, "Metric counter value is not a number", http.StatusBadRequest)
			return
		}
		m.metricHandler.AddCounter(mname, mvalue)
	default:
		http.Error(w, "Metric type is not supported", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
