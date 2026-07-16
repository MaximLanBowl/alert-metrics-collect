package repository

import models "github.com/MaximLanBowl/alert-metrics-collect/internal/model"

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) SetGauge(name string, value float64) {
	m.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
}

func (m *MemStorage) AddCounter(name string, delta int64) {
	val, ok := m.metrics[name]
	if ok {
		delta += *val.Delta
	}
	m.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}
}
