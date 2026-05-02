package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
)

type Pool struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, cfg config.Config) (*Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not configured")
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	wrapped := &Pool{pool: pool}
	if err := wrapped.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return wrapped, nil
}

func (p *Pool) Raw() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.pool
}

func (p *Pool) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return errors.New("postgres pool is not configured")
	}
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

type Checker struct {
	Pool *Pool
}

func (c Checker) Name() string { return "postgres" }

func (c Checker) Check(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}
