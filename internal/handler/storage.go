package handler

import (
	"context"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/models"
)

type MetricsStorage interface {
	SetGauge(ctx context.Context, name string, value float64) error
	SetCounter(ctx context.Context, name string, delta int64) error
	UpdateMetrics(ctx context.Context, req models.Metrics) error
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error

	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAll(ctx context.Context, filter models.MetricsFilter) ([]models.Metrics, error)
	GetByValues(ctx context.Context, req models.Metrics) (models.Metrics, error)
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
