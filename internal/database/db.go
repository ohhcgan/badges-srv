package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" /* registers "pgx" driver for database/sql */
)

type DBConfig struct {
	MinConns              int32
	MaxConns              int32
	MaxConnIdleTime       time.Duration
	MaxConnLifetime       time.Duration
	HealthCheckPeriod     time.Duration
	MaxConnLifetimeJitter time.Duration
}

func NewPool(ctx context.Context, connStr string, dbConf *DBConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parsing connStr: %w", err)
	}

	cfg.MinConns = dbConf.MinConns
	cfg.MaxConns = dbConf.MaxConns
	cfg.MaxConnIdleTime = dbConf.MaxConnIdleTime
	cfg.MaxConnLifetime = dbConf.MaxConnLifetime
	cfg.HealthCheckPeriod = dbConf.HealthCheckPeriod
	cfg.MaxConnLifetimeJitter = dbConf.MaxConnLifetimeJitter

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pgxpool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return pool, nil
}
