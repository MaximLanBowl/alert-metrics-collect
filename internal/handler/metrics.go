package handler

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	models "github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

//go:embed templates/index.html
var templateFS embed.FS
var indexTmp = template.Must(template.ParseFS(templateFS, "templates/index.html"))

type MetricsStorage interface {
	MetricsReader
	MetricsWriter
	MetricsRestore
}

type MetricsReader interface {
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() []models.Metrics
	GetByValues(req models.Metrics) (models.Metrics, error)
}

type MetricsWriter interface {
	SetGauge(name string, value float64)
	AddCounter(name string, delta int64)
	UpdateMetrics(req models.Metrics)
}

type MetricsRestore interface {
	Restore(metrics []models.Metrics) error
}

type MetricsProducer interface {
	WriteMetrics(metrics []models.Metrics) error
	Close() error
}

type MetricsConsumer interface {
	ReadMetrics() ([]models.Metrics, error)
	Close() error
}

type MetricsHandler struct {
	metricHandler MetricsStorage
	producer      MetricsProducer
	cfg           config.ServerConfig
}

func NewMetrics(metricHandler MetricsStorage, producer MetricsProducer, cfg config.ServerConfig) *MetricsHandler {
	return &MetricsHandler{
		metricHandler: metricHandler,
		producer:      producer,
		cfg:           cfg,
	}
}

func (m *MetricsHandler) CloseMetricsFile() error {
	return m.producer.Close()
}

func (m *MetricsHandler) Restore(consumer MetricsConsumer) error {
	if !m.cfg.Restore {
		return nil
	}

	metrics, err := consumer.ReadMetrics()
	if err != nil {
		log.Warn().Err(err).Msg("Error reading metrics")
		return err
	}

	m.metricHandler.Restore(metrics)

	return nil
}

func (m *MetricsHandler) StartAutoSave() {
	if m.cfg.StoreInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Duration(m.cfg.StoreInterval) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := m.producer.WriteMetrics(m.metricHandler.GetAll()); err != nil {
				log.Warn().Err(err).Msg("failed to write metrics")
			}
		}
	}()
}

func (m *MetricsHandler) syncSave() {
	if m.cfg.StoreInterval != 0 {
		return
	}

	if err := m.producer.WriteMetrics(m.metricHandler.GetAll()); err != nil {
		log.Warn().Err(err).Msg("failed to write metrics")
		return
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

	m.syncSave()

	log.Info().Msgf("Metric %s set to %s", mname, mvalue)
	w.WriteHeader(http.StatusOK)
}

func (m *MetricsHandler) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	if !methodCheck(w, r) {
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msg("Error decoding body")
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	if !typeCheck(req, w) {
		return
	}

	if req.MType == models.Gauge && req.Value == nil {
		log.Warn().Msg("Invalid request value gauge")
		http.Error(w, "invalid request value gauge", http.StatusBadRequest)
		return
	}

	if req.MType == models.Counter && req.Delta == nil {
		log.Warn().Msg("Invalid request value counter")
		http.Error(w, "invalid request value counter", http.StatusBadRequest)
		return
	}

	m.metricHandler.UpdateMetrics(req)
	m.syncSave()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(req); err != nil {
		log.Warn().Err(err).Msg("Error encoding body")
		http.Error(w, "failed to encode request body", http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Metrics updated successfully")
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := indexTmp.Execute(w, metrics); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (m *MetricsHandler) GetMetricsByValue(w http.ResponseWriter, r *http.Request) {
	if !methodCheck(w, r) {
		return
	}

	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msgf("Error decoding body")
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	if !typeCheck(req, w) {
		return
	}

	data, err := m.metricHandler.GetByValues(req)
	if err != nil {
		log.Warn().Err(err).Msg("Error getting metric by value")
		http.Error(w, "failed to get metric by value", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode request body", http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Got metric by value successfully")
}

func typeCheck(req models.Metrics, w http.ResponseWriter) bool {
	if req.MType != models.Gauge && req.MType != models.Counter {
		log.Warn().Msg("Invalid metric type")
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return false
	}

	return true
}

func methodCheck(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		log.Warn().Msg("Invalid method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	return true
}
