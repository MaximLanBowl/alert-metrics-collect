package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Postgres PostgresConfig
	Server   ServerConfig
}

func Load(flags Flags) (Config, error) {
	postgres, err := loadPostgres(flags)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load postgres config: %w", err)
	}

	server, err := loadServer(flags)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load server config: %w", err)
	}

	return Config{
		Postgres: postgres,
		Server:   server,
	}, nil
}

type PostgresConfig struct {
	DatabaseDSN string `env:"DATABASE_DSN"`
}

func loadPostgres(flags Flags) (PostgresConfig, error) {
	cfg := PostgresConfig{
		DatabaseDSN: flags.DatabaseDSN,
	}

	if err := env.Parse(&cfg); err != nil {
		return PostgresConfig{}, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	return cfg, nil
}

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func loadServer(flags Flags) (ServerConfig, error) {
	cfg := ServerConfig{
		Address:         flags.Address,
		StoreInterval:   flags.StoreInterval,
		FileStoragePath: flags.FileStoragePath,
		Restore:         flags.Restore,
	}

	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to parse server config: %w", err)
	}

	return cfg, nil
}
