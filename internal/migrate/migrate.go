package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

const schemaTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  name text NOT NULL,
  checksum text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now()
);`

type Migration struct {
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

func LoadDir(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		base := strings.TrimSuffix(entry.Name(), ".sql")
		version := base
		name := base
		if idx := strings.Index(base, "_"); idx > 0 {
			version = base[:idx]
			name = base[idx+1:]
		}
		checksumBytes := sha256.Sum256(content)
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Path:     path,
			Checksum: hex.EncodeToString(checksumBytes[:]),
			SQL:      string(content),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func (r *Runner) Up(ctx context.Context) ([]Migration, error) {
	if r.pool == nil || r.pool.Raw() == nil {
		return nil, errors.New("postgres pool is not configured")
	}
	migrations, err := LoadDir(r.dir)
	if err != nil {
		return nil, err
	}
	if _, err := r.pool.Raw().Exec(ctx, schemaTableSQL); err != nil {
		return nil, fmt.Errorf("ensure schema migrations table: %w", err)
	}
	applied, err := r.applied(ctx)
	if err != nil {
		return nil, err
	}
	appliedNow := make([]Migration, 0)
	for _, migration := range migrations {
		if checksum, ok := applied[migration.Version]; ok {
			if checksum != migration.Checksum {
				return nil, fmt.Errorf("migration %s checksum changed", migration.Version)
			}
			continue
		}
		if err := r.applyOne(ctx, migration); err != nil {
			return nil, err
		}
		appliedNow = append(appliedNow, migration)
	}
	return appliedNow, nil
}

func (r *Runner) applied(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Raw().Query(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func (r *Runner) applyOne(ctx context.Context, migration Migration) error {
	tx, err := r.pool.Raw().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("execute migration %s: %w", migration.Version, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
		migration.Version,
		migration.Name,
		migration.Checksum,
	); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Version, err)
	}
	return nil
}
