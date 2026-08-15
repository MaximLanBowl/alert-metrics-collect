package repository

import (
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

func (m *MemStorage) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
}

func (m *MemStorage) AddCounter(name string, delta int64) {
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
}

func (m *MemStorage) UpdateMetrics(req models.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *MemStorage) GetByValues(req models.Metrics) (models.Metrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.metrics[req.ID]
	if !ok {
		return models.Metrics{}, errors.New("metric not found")
	}

	return val, nil
}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
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

func (m *MemStorage) GetCounter(name string) (int64, bool) {
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

func (m *MemStorage) GetAll() []models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := make([]models.Metrics, 0, len(m.metrics))
	for _, v := range m.metrics {
		values = append(values, v)
	}

	return values
}

func (m *MemStorage) Restore(metrics []models.Metrics) error {
	for _, mt := range metrics {
		m.UpdateMetrics(mt)
	}

	return nil
}
