package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/db"
	"github.com/MaximLanBowl/alert-metrics-collect/internal/models"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type MetricsDB struct {
	pool *pgxpool.Pool
}

func NewDBStorage(pool *pgxpool.Pool) *MetricsDB {
	return &MetricsDB{
		pool: pool,
	}
}

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func (m *MetricsDB) SetGauge(ctx context.Context, name string, value float64) error {
	b := m.buildGaugeQuery(name, value)

	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build sql query gauge: %w", err)
	}

	_, err = m.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute sql query gauge: %w", err)
	}

	return nil
}

func (m *MetricsDB) SetCounter(ctx context.Context, name string, delta int64) error {
	b := m.buildCounterQuery(name, delta)

	query, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build sql query counter: %w", err)
	}

	_, err = m.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute sql query counter: %w", err)
	}

	return nil
}

func (m *MetricsDB) UpdateMetrics(ctx context.Context, req models.Metrics) error {
	return db.WithRetry(func() error {
		return m.updateMetrics(ctx, req)
	})
}

func (m *MetricsDB) updateMetrics(ctx context.Context, req models.Metrics) error {
	switch req.MType {
	case models.Gauge:
		if req.Value == nil {
			return fmt.Errorf("value is required for gauge")
		}
		if err := m.SetGauge(ctx, req.ID, *req.Value); err != nil {
			return fmt.Errorf("failed to set gauge: %w", err)
		}
	case models.Counter:
		if req.Delta == nil {
			return fmt.Errorf("delta is required for counter")
		}
		if err := m.SetCounter(ctx, req.ID, *req.Delta); err != nil {
			return fmt.Errorf("failed to set counter: %w", err)
		}
	default:
		return fmt.Errorf("invalid metric type: %s", req.MType)
	}

	return nil
}

func (m *MetricsDB) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	return db.WithRetry(func() error {
		return m.updateMetricsBatch(ctx, metrics)
	})
}

func (m *MetricsDB) updateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := pgx.Batch{}

	for _, mt := range metrics {
		switch mt.MType {
		case models.Gauge:
			if mt.Value == nil {
				continue
			}
			b := m.buildGaugeQuery(mt.ID, *mt.Value)
			query, args, err := b.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build sql query metrics: %w", err)
			}
			batch.Queue(query, args...)
		case models.Counter:
			if mt.Delta == nil {
				continue
			}
			b := m.buildCounterQuery(mt.ID, *mt.Delta)
			query, args, err := b.ToSql()
			if err != nil {
				return fmt.Errorf("failed to build sql query metrics: %w", err)
			}
			batch.Queue(query, args...)
		}
	}

	if batch.Len() == 0 {
		log.Warn().Msg("EMPTY BATCH")
		return nil
	}

	bs := tx.SendBatch(ctx, &batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := bs.Exec(); err != nil {
			err = bs.Close()
			if err != nil {
				return fmt.Errorf("failed to close batch in bs.Exec: %w", err)
			}

			return fmt.Errorf("failed to execute sql query metrics: %w", err)
		}
	}

	if err = bs.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	return tx.Commit(ctx)
}

func (m *MetricsDB) GetAll(ctx context.Context, mtf models.MetricsFilter) ([]models.Metrics, error) {
	b := psql.Select(
		"id",
		"mtype",
		"delta",
		"value",
	).From("metrics").OrderBy("id ASC")

	b = m.filter(b, mtf)

	query, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build sql query metrics: %w", err)
	}

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sql query metrics: %w", err)
	}
	defer rows.Close()

	metrics := make([]models.Metrics, 0)
	for rows.Next() {
		mt := models.Metrics{}
		err = rows.Scan(
			&mt.ID,
			&mt.MType,
			&mt.Delta,
			&mt.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		metrics = append(metrics, mt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}

	return metrics, nil
}

func (m *MetricsDB) GetGauge(ctx context.Context, name string) (float64, bool) {
	metrics, err := m.GetAll(ctx, models.MetricsFilter{
		ID:    name,
		MType: models.Gauge,
	})
	if err != nil || len(metrics) == 0 || metrics[0].Value == nil {
		return 0, false
	}

	return *metrics[0].Value, true
}

func (m *MetricsDB) GetCounter(ctx context.Context, name string) (int64, bool) {
	metrics, err := m.GetAll(ctx, models.MetricsFilter{
		ID:    name,
		MType: models.Counter,
	})
	if err != nil || len(metrics) == 0 || metrics[0].Delta == nil {
		return 0, false
	}

	return *metrics[0].Delta, true
}

func (m *MetricsDB) GetByValues(ctx context.Context, req models.Metrics) (models.Metrics, error) {
	metrics, err := m.GetAll(ctx, models.MetricsFilter{
		ID:    req.ID,
		MType: req.MType,
	})
	if err != nil {
		return models.Metrics{}, err
	}

	if len(metrics) == 0 {
		return models.Metrics{}, errors.New("metric not found")
	}

	return metrics[0], nil
}

func (m *MetricsDB) filter(b sq.SelectBuilder, f models.MetricsFilter) sq.SelectBuilder {
	if f.ID != "" {
		b = b.Where(sq.Eq{"id": f.ID})
	}

	if f.MType != "" {
		b = b.Where(sq.Eq{"mtype": f.MType})
	}

	return b
}

func (m *MetricsDB) buildGaugeQuery(name string, value float64) sq.InsertBuilder {
	b := psql.Insert("metrics").
		Columns(
			"id",
			"mtype",
			"value",
		).
		Values(
			name,
			models.Gauge,
			value,
		).
		Suffix(
			`ON CONFLICT (id) DO UPDATE SET
			mtype = EXCLUDED.mtype,
			value = EXCLUDED.value,
			delta = NULL`,
		)

	return b
}

func (m *MetricsDB) buildCounterQuery(name string, delta int64) sq.InsertBuilder {
	b := psql.Insert("metrics").
		Columns(
			"id",
			"mtype",
			"delta",
		).
		Values(
			name,
			models.Counter,
			delta,
		).
		Suffix(
			`ON CONFLICT (id) DO UPDATE SET
			mtype = EXCLUDED.mtype,
			delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
			value = NULL`,
		)

	return b
}
