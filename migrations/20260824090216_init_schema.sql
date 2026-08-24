-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    id TEXT PRIMARY KEY,
    mtype TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    hash TEXT
);

-- +goose Down
DROP TABLE IF EXISTS metrics;
