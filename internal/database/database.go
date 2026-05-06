package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	_ "modernc.org/sqlite"
)

type Driver string

const (
	DriverPostgres Driver = "postgres"
	DriverSQLite   Driver = "sqlite"
)

type Pool struct {
	driver   Driver
	postgres *pgxpool.Pool
	sqlite   *sql.DB
}

func Open(ctx context.Context, cfg config.Config) (*Pool, error) {
	switch Driver(cfg.DBDriver) {
	case DriverSQLite:
		return openSQLite(ctx, cfg)
	case "", DriverPostgres:
		return openPostgres(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s", cfg.DBDriver)
	}
}

func openPostgres(ctx context.Context, cfg config.Config) (*Pool, error) {
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
	wrapped := &Pool{driver: DriverPostgres, postgres: pool}
	if err := wrapped.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return wrapped, nil
}

func openSQLite(ctx context.Context, cfg config.Config) (*Pool, error) {
	if cfg.SQLitePath == "" {
		return nil, errors.New("SQLITE_PATH is not configured")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o750); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure sqlite pragma %q: %w", pragma, err)
		}
	}
	wrapped := &Pool{driver: DriverSQLite, sqlite: db}
	if err := wrapped.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return wrapped, nil
}

func (p *Pool) Driver() Driver {
	if p == nil {
		return ""
	}
	return p.driver
}

func (p *Pool) Raw() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.postgres
}

func (p *Pool) SQLDB() *sql.DB {
	if p == nil {
		return nil
	}
	return p.sqlite
}

func (p *Pool) Close() {
	if p == nil {
		return
	}
	if p.postgres != nil {
		p.postgres.Close()
	}
	if p.sqlite != nil {
		_ = p.sqlite.Close()
	}
}

func (p *Pool) Ping(ctx context.Context) error {
	if p == nil {
		return errors.New("database pool is not configured")
	}
	switch p.driver {
	case DriverPostgres:
		if p.postgres == nil {
			return errors.New("postgres pool is not configured")
		}
		if err := p.postgres.Ping(ctx); err != nil {
			return fmt.Errorf("ping postgres: %w", err)
		}
	case DriverSQLite:
		if p.sqlite == nil {
			return errors.New("sqlite database is not configured")
		}
		if err := p.sqlite.PingContext(ctx); err != nil {
			return fmt.Errorf("ping sqlite: %w", err)
		}
	default:
		return errors.New("unknown database driver")
	}
	return nil
}

func (p *Pool) IsPostgres() bool {
	return p != nil && p.driver == DriverPostgres && p.postgres != nil
}

func (p *Pool) IsSQLite() bool {
	return p != nil && p.driver == DriverSQLite && p.sqlite != nil
}

type Checker struct {
	Pool *Pool
}

func (c Checker) Name() string {
	if c.Pool == nil || c.Pool.Driver() == "" {
		return "database"
	}
	return string(c.Pool.Driver())
}

func (c Checker) Check(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}
