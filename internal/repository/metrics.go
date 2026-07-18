package repository

import (
	"sync"

	models "github.com/MaximLanBowl/alert-metrics-collect/internal/model"
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
