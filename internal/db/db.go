package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Init(cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	connPool, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create db pool connection: %w", err)
	}

	if err = connPool.Ping(ctx); err != nil {
		connPool.Close()
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return connPool, nil
}
