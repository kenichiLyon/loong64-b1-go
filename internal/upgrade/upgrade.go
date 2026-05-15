package upgrade

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

const ScopeDatabase = "database"

const postgresJournalTableSQL = `
CREATE TABLE IF NOT EXISTS system_upgrades (
  scope text NOT NULL,
  version text NOT NULL,
  name text NOT NULL,
  checksum text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (scope, version)
);`

const sqliteJournalTableSQL = `
CREATE TABLE IF NOT EXISTS system_upgrades (
  scope text NOT NULL,
  version text NOT NULL,
  name text NOT NULL,
  checksum text NOT NULL,
  applied_at text NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (scope, version)
);`

// Step is one idempotent system upgrade unit. Database SQL files are only one
// scope of upgrade; future scopes can cover storage, config, workers, and data
// backfills without being owned by the database package.
type Step struct {
	Scope    string
	Version  string
	Name     string
	Path     string
	Checksum string
	SQL      string
}

type Runner struct {
	pool *database.Pool
	dir  string
}

func NewRunner(pool *database.Pool, dir string) *Runner {
	if dir == "" {
		dir = "migrations"
	}
	return &Runner{pool: pool, dir: dir}
}

func LoadDir(dir string) ([]Step, error) {
	return LoadSQLSteps(dir, ScopeDatabase)
}

func LoadSQLSteps(dir, scope string) ([]Step, error) {
	if strings.TrimSpace(scope) == "" {
		scope = ScopeDatabase
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read upgrade dir: %w", err)
	}
	steps := make([]Step, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read upgrade step %s: %w", entry.Name(), err)
		}
		base := strings.TrimSuffix(entry.Name(), ".sql")
		version := base
		name := base
		if idx := strings.Index(base, "_"); idx > 0 {
			version = base[:idx]
			name = base[idx+1:]
		}
		checksumBytes := sha256.Sum256(content)
		steps = append(steps, Step{
			Scope:    scope,
			Version:  version,
			Name:     name,
			Path:     path,
			Checksum: hex.EncodeToString(checksumBytes[:]),
			SQL:      string(content),
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].Scope != steps[j].Scope {
			return steps[i].Scope < steps[j].Scope
		}
		return steps[i].Version < steps[j].Version
	})
	return steps, nil
}

func (r *Runner) Up(ctx context.Context) ([]Step, error) {
	if r.pool == nil {
		return nil, errors.New("database pool is not configured for system upgrade journal")
	}
	steps, err := LoadSQLSteps(r.databaseUpgradeDir(), ScopeDatabase)
	if err != nil {
		return nil, err
	}
	switch r.pool.Driver() {
	case database.DriverPostgres:
		return r.upPostgres(ctx, steps)
	case database.DriverSQLite:
		return r.upSQLite(ctx, steps)
	default:
		return nil, errors.New("unknown database driver")
	}
}

func (r *Runner) databaseUpgradeDir() string {
	if r.pool != nil && r.pool.Driver() == database.DriverSQLite {
		return filepath.Join(r.dir, "sqlite")
	}
	return r.dir
}

func (r *Runner) upPostgres(ctx context.Context, steps []Step) ([]Step, error) {
	if r.pool == nil || r.pool.Raw() == nil {
		return nil, errors.New("postgres pool is not configured")
	}
	if _, err := r.pool.Raw().Exec(ctx, postgresJournalTableSQL); err != nil {
		return nil, fmt.Errorf("ensure system upgrade journal: %w", err)
	}
	if err := r.importLegacyPostgresJournal(ctx); err != nil {
		return nil, err
	}
	applied, err := r.appliedPostgres(ctx, ScopeDatabase)
	if err != nil {
		return nil, err
	}
	appliedNow := make([]Step, 0)
	for _, step := range steps {
		if checksum, ok := applied[step.Version]; ok {
			if checksum != step.Checksum {
				return nil, fmt.Errorf("upgrade step %s/%s checksum changed", step.Scope, step.Version)
			}
			continue
		}
		if err := r.applyOnePostgres(ctx, step); err != nil {
			return nil, err
		}
		appliedNow = append(appliedNow, step)
	}
	return appliedNow, nil
}

func (r *Runner) importLegacyPostgresJournal(ctx context.Context) error {
	var exists bool
	if err := r.pool.Raw().QueryRow(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect legacy migration journal: %w", err)
	}
	if !exists {
		return nil
	}
	if _, err := r.pool.Raw().Exec(ctx, `
INSERT INTO system_upgrades (scope, version, name, checksum)
SELECT $1, version, name, checksum FROM schema_migrations
ON CONFLICT (scope, version) DO NOTHING`, ScopeDatabase); err != nil {
		return fmt.Errorf("import legacy migration journal: %w", err)
	}
	return nil
}

func (r *Runner) appliedPostgres(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := r.pool.Raw().Query(ctx, `SELECT version, checksum FROM system_upgrades WHERE scope = $1`, scope)
	if err != nil {
		return nil, fmt.Errorf("query applied upgrade steps: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("scan upgrade step: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func (r *Runner) applyOnePostgres(ctx context.Context, step Step) error {
	tx, err := r.pool.Raw().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, step.SQL); err != nil {
		return fmt.Errorf("execute upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO system_upgrades (scope, version, name, checksum) VALUES ($1, $2, $3, $4)`,
		step.Scope,
		step.Version,
		step.Name,
		step.Checksum,
	); err != nil {
		return fmt.Errorf("record upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	return nil
}

func (r *Runner) upSQLite(ctx context.Context, steps []Step) ([]Step, error) {
	if r.pool == nil || r.pool.SQLDB() == nil {
		return nil, errors.New("sqlite database is not configured")
	}
	if _, err := r.pool.SQLDB().ExecContext(ctx, sqliteJournalTableSQL); err != nil {
		return nil, fmt.Errorf("ensure system upgrade journal: %w", err)
	}
	if err := r.importLegacySQLiteJournal(ctx); err != nil {
		return nil, err
	}
	applied, err := r.appliedSQLite(ctx, ScopeDatabase)
	if err != nil {
		return nil, err
	}
	appliedNow := make([]Step, 0)
	for _, step := range steps {
		if checksum, ok := applied[step.Version]; ok {
			if checksum != step.Checksum {
				return nil, fmt.Errorf("upgrade step %s/%s checksum changed", step.Scope, step.Version)
			}
			continue
		}
		if err := r.applyOneSQLite(ctx, step); err != nil {
			return nil, err
		}
		appliedNow = append(appliedNow, step)
	}
	return appliedNow, nil
}

func (r *Runner) importLegacySQLiteJournal(ctx context.Context) error {
	var name string
	err := r.pool.SQLDB().QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect legacy migration journal: %w", err)
	}
	if _, err := r.pool.SQLDB().ExecContext(ctx, `
INSERT INTO system_upgrades (scope, version, name, checksum)
SELECT ?, version, name, checksum FROM schema_migrations
ON CONFLICT(scope, version) DO NOTHING`, ScopeDatabase); err != nil {
		return fmt.Errorf("import legacy migration journal: %w", err)
	}
	return nil
}

func (r *Runner) appliedSQLite(ctx context.Context, scope string) (map[string]string, error) {
	rows, err := r.pool.SQLDB().QueryContext(ctx, `SELECT version, checksum FROM system_upgrades WHERE scope = ?`, scope)
	if err != nil {
		return nil, fmt.Errorf("query applied upgrade steps: %w", err)
	}
	defer func() { _ = rows.Close() }()
	applied := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("scan upgrade step: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func (r *Runner) applyOneSQLite(ctx context.Context, step Step) error {
	tx, err := r.pool.SQLDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, step.SQL); err != nil {
		return fmt.Errorf("execute upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO system_upgrades (scope, version, name, checksum) VALUES (?, ?, ?, ?)`,
		step.Scope,
		step.Version,
		step.Name,
		step.Checksum,
	); err != nil {
		return fmt.Errorf("record upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upgrade step %s/%s: %w", step.Scope, step.Version, err)
	}
	return nil
}
