package repository

import (
	"context"
	"errors"
	"sync"

	models "github.com/MaximLanBowl/alert-metrics-collect/internal/models"
)

type MemStorage struct {
	mu      sync.RWMutex
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mu:      sync.RWMutex{},
		metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) SetGauge(_ context.Context, name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}

	return nil
}

func (m *MemStorage) SetCounter(_ context.Context, name string, delta int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.metrics[name]
	if ok && val.Delta != nil {
		delta += *val.Delta
	}

	m.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}

	return nil
}

func (m *MemStorage) updateMetricsLocked(req models.Metrics) {
	switch req.MType {
	case models.Counter:
		existing, ok := m.metrics[req.ID]
		if ok && existing.Delta != nil && req.Delta != nil {
			*req.Delta += *existing.Delta
		}
		m.metrics[req.ID] = req
	case models.Gauge:
		m.metrics[req.ID] = req
	}
}

func (m *MemStorage) UpdateMetrics(_ context.Context, req models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateMetricsLocked(req)

	return nil
}

func (m *MemStorage) GetByValues(_ context.Context, req models.Metrics) (models.Metrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.metrics[req.ID]
	if !ok {
		return models.Metrics{}, errors.New("metric not found")
	}

	return val, nil
}

func (m *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.metrics[name]
	if ok {
		if val.Value != nil && val.MType == models.Gauge {
			return *val.Value, true
		}
		return 0, false
	}

	return 0, false
}

func (m *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.metrics[name]
	if ok {
		if val.Delta != nil && val.MType == models.Counter {
			return *val.Delta, true
		}
		return 0, false
	}

	return 0, false
}

func (m *MemStorage) GetAll(_ context.Context, _ models.MetricsFilter) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := make([]models.Metrics, 0, len(m.metrics))
	for _, v := range m.metrics {
		values = append(values, v)
	}

	return values, nil
}

func (m *MemStorage) Restore(metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mt := range metrics {
		m.updateMetricsLocked(mt)
	}

	return nil
}

func (m *MemStorage) UpdateMetricsBatch(_ context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mt := range metrics {
		m.updateMetricsLocked(mt)
	}

	return nil
}
