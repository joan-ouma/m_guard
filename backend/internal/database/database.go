package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Error("Database configuration parse failed", "error", err)
		return nil, fmt.Errorf("parse config error: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	slog.Info("Connecting to database", "host", config.ConnConfig.Host, "port", config.ConnConfig.Port, "database", config.ConnConfig.Database, "user", config.ConnConfig.User)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("Database pool creation failed", "error", err)
		return nil, fmt.Errorf("pool connect error: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		slog.Error("Database ping failed", "error", err)
		return nil, fmt.Errorf("ping error: %w", err)
	}

	slog.Info("Database connected successfully", "host", config.ConnConfig.Host, "database", config.ConnConfig.Database)

	return pool, nil
}
