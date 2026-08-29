-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    id VARCHAR(50) PRIMARY KEY,
    mtype VARCHAR(50) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION
);

-- +goose Down
DROP TABLE IF EXISTS metrics;
