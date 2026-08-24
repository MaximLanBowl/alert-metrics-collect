package handler

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

//go:embed templates/index.html
var templateFS embed.FS

var indexTmp = template.Must(
	template.ParseFS(templateFS, "templates/index.html"),
)

type MetricsHandler struct {
	metricHandler MetricsStorage
	producer      MetricsProducer
	cfg           config.ServerConfig
}

func NewMetrics(
	metricHandler MetricsStorage,
	producer MetricsProducer,
	cfg config.ServerConfig,
) *MetricsHandler {
	return &MetricsHandler{
		metricHandler: metricHandler,
		producer:      producer,
		cfg:           cfg,
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

	var err error

	switch mtype {
	case models.Gauge:
		value, parseErr := strconv.ParseFloat(mvalue, 64)
		if parseErr != nil {
			http.Error(w, "gauge value is not a number", http.StatusBadRequest)
			return
		}

		err = m.metricHandler.SetGauge(r.Context(), mname, value)

	case models.Counter:
		value, parseErr := strconv.ParseInt(mvalue, 10, 64)
		if parseErr != nil {
			http.Error(w, "counter value is not a number", http.StatusBadRequest)
			return
		}

		err = m.metricHandler.SetCounter(r.Context(), mname, value)

	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Warn().Err(err).Msg("failed to set metric")
		http.Error(w, "failed to set metric", http.StatusInternalServerError)
		return
	}

	m.syncSave()

	w.WriteHeader(http.StatusOK)
}

func (m *MetricsHandler) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	if !methodCheck(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req models.Metrics
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	if !typeCheck(req, w) {
		return
	}

	if req.MType == models.Gauge && req.Value == nil {
		http.Error(w, "invalid request value gauge", http.StatusBadRequest)
		return
	}

	if req.MType == models.Counter && req.Delta == nil {
		http.Error(w, "invalid request value counter", http.StatusBadRequest)
		return
	}

	if err = m.metricHandler.UpdateMetrics(r.Context(), req); err != nil {
		log.Warn().Err(err).Msg("failed to update metrics")
		http.Error(w, "failed to update metrics", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "failed to marshal request body", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(resp)
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
		value, ok := m.metricHandler.GetGauge(r.Context(), mname)
		if !ok {
			http.Error(w, "gauge metric not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(
			strconv.FormatFloat(value, 'f', -1, 64),
		))

	case models.Counter:
		value, ok := m.metricHandler.GetCounter(r.Context(), mname)
		if !ok {
			http.Error(w, "counter metric not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(
			strconv.FormatInt(value, 10),
		))

	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
	}
}

func (m *MetricsHandler) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := m.metricHandler.GetAll(
		r.Context(),
		models.MetricsFilter{},
	)
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	var buf bytes.Buffer

	if err := indexTmp.Execute(&buf, metrics); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(buf.Bytes())
}

func (m *MetricsHandler) GetMetricsByValue(w http.ResponseWriter, r *http.Request) {
	if !methodCheck(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req models.Metrics
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	if !typeCheck(req, w) {
		return
	}

	data, err := m.metricHandler.GetByValues(r.Context(), req)
	if err != nil {
		http.Error(w, "failed to get metric by value", http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(resp)
}

func (m *MetricsHandler) Restore(consumer MetricsConsumer) error {
	if !m.cfg.Restore {
		return nil
	}

	storage, ok := m.metricHandler.(MetricsRestore)
	if !ok {
		return nil
	}

	metrics, err := consumer.ReadMetrics()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return fmt.Errorf("failed to read metrics: %w", err)
	}

	if err := storage.Restore(metrics); err != nil {
		return fmt.Errorf("failed to restore metrics: %w", err)
	}

	return nil
}

func (m *MetricsHandler) syncSave() {
	if m.producer == nil || m.cfg.StoreInterval != 0 {
		return
	}

	metrics, err := m.metricHandler.GetAll(
		context.Background(),
		models.MetricsFilter{},
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get metrics")
		return
	}

	if err := m.producer.WriteMetrics(metrics); err != nil {
		log.Warn().Err(err).Msg("failed to write metrics")
	}
}

func (m *MetricsHandler) CloseMetricsFile() error {
	if m.producer == nil {
		return nil
	}

	return m.producer.Close()
}

func (m *MetricsHandler) StartAutoSave() {
	if m.producer == nil || m.cfg.StoreInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(
			time.Duration(m.cfg.StoreInterval) * time.Second,
		)
		defer ticker.Stop()

		for range ticker.C {
			metrics, err := m.metricHandler.GetAll(
				context.Background(),
				models.MetricsFilter{},
			)
			if err != nil {
				log.Warn().Err(err).Msg("failed to get metrics")
				continue
			}

			if err := m.producer.WriteMetrics(metrics); err != nil {
				log.Warn().Err(err).Msg("failed to write metrics")
			}
		}
	}()
}

func typeCheck(req models.Metrics, w http.ResponseWriter) bool {
	if req.MType != models.Gauge && req.MType != models.Counter {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return false
	}

	return true
}

func methodCheck(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	return true
}
