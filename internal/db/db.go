package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/config"
	"github.com/MaximLanBowl/alert-metrics-collect/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Init(cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	if cfg.DatabaseDSN == "" {
		return nil, fmt.Errorf("database DSN is empty (set -d flag or DATABASE_DSN env)")
	}

	if err := runMigrations(cfg.DatabaseDSN); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

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

func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db connection: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)

	if err = goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err = goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
