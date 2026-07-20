package handler

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"

	models "github.com/MaximLanBowl/alert-metrics-collect/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

//go:embed templates/index.html
var templateFS embed.FS

var indexTmp = template.Must(template.ParseFS(templateFS, "templates/index.html"))

type MetricsStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() []models.Metrics
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
	mtype := chi.URLParam(r, "type")
	mname := chi.URLParam(r, "name")
	mvalue := chi.URLParam(r, "value")

	if mname == "" {
		http.Error(w, "invalid metric name", http.StatusNotFound)
		return
	}

	switch mtype {
	case models.Gauge:
		gaugeVal, err := strconv.ParseFloat(mvalue, 64)
		if err != nil {
			http.Error(w, "gauge value is not a number", http.StatusBadRequest)
			return
		}
		m.metricHandler.SetGauge(mname, gaugeVal)
	case models.Counter:
		counterVal, err := strconv.ParseInt(mvalue, 10, 64)
		if err != nil {
			http.Error(w, "counter value is not a number", http.StatusBadRequest)
			return
		}
		m.metricHandler.AddCounter(mname, counterVal)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	log.Info().Msgf("Metric %s set to %s", mname, mvalue)
	w.WriteHeader(http.StatusOK)
}

func (m *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	mtype := chi.URLParam(r, "type")
	mname := chi.URLParam(r, "name")

	if mname == "" {
		http.Error(w, "invalid metric name", http.StatusNotFound)
		return
	}

	switch mtype {
	case models.Gauge:
		gaugeVal, ok := m.metricHandler.GetGauge(mname)
		if !ok {
			http.Error(w, "gauge metric not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatFloat(gaugeVal, 'f', -1, 64)))
	case models.Counter:
		counterVal, ok := m.metricHandler.GetCounter(mname)
		if !ok {
			http.Error(w, "counter metric not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatInt(counterVal, 10)))
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (m *MetricsHandler) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics := m.metricHandler.GetAll()

	w.Header().Set("Content-Type", "text/html, charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err := indexTmp.Execute(w, metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
