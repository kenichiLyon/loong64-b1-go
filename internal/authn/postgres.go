package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type PostgresRepository struct {
	db *database.Pool
}

func NewPostgresRepository(db *database.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

type pgxPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (r *PostgresRepository) pool() (pgxPool, error) {
	if r == nil || r.db == nil || r.db.Raw() == nil {
		return nil, unavailableError("auth postgres repository is not configured", nil)
	}
	return r.db.Raw(), nil
}

func (r *PostgresRepository) GetUserAuthByUsername(ctx context.Context, username string) (UserAuth, error) {
	pool, err := r.pool()
	if err != nil {
		return UserAuth{}, err
	}
	return loadUserAuth(ctx, pool, `SELECT id, username, display_name, status, COALESCE(password_hash, '') AS password_hash FROM users WHERE lower(username)=lower($1)`, username)
}

func (r *PostgresRepository) GetUserAuthByID(ctx context.Context, id string) (UserAuth, error) {
	pool, err := r.pool()
	if err != nil {
		return UserAuth{}, err
	}
	return loadUserAuth(ctx, pool, `SELECT id, username, display_name, status, COALESCE(password_hash, '') AS password_hash FROM users WHERE id=$1`, id)
}

func loadUserAuth(ctx context.Context, pool pgxPool, query string, arg string) (UserAuth, error) {
	var user UserAuth
	if err := pool.QueryRow(ctx, query, arg).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Status, &user.PasswordHash); err != nil {
		return UserAuth{}, mapDBError(err)
	}
	rows, err := pool.Query(ctx, `SELECT role FROM user_roles WHERE user_id = $1 ORDER BY role`, user.ID)
	if err != nil {
		return UserAuth{}, mapDBError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return UserAuth{}, mapDBError(err)
		}
		user.Roles = append(user.Roles, teaching.Role(role))
	}
	if err := rows.Err(); err != nil {
		return UserAuth{}, mapDBError(err)
	}
	return user, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session Session) (Session, error) {
	pool, err := r.pool()
	if err != nil {
		return Session{}, err
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO auth_sessions (id, user_id, token_hash, expires_at, last_seen_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, expires_at, created_at, last_seen_at`,
		session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.LastSeenAt).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt); err != nil {
		return Session{}, mapDBError(err)
	}
	session.User, err = r.GetUserAuthByID(ctx, session.UserID)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (r *PostgresRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	pool, err := r.pool()
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := pool.QueryRow(ctx, `
SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at
FROM auth_sessions
WHERE token_hash = $1`, tokenHash).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt); err != nil {
		return Session{}, mapDBError(err)
	}
	session.User, err = r.GetUserAuthByID(ctx, session.UserID)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (r *PostgresRepository) TouchSession(ctx context.Context, session Session) (Session, error) {
	pool, err := r.pool()
	if err != nil {
		return Session{}, err
	}
	if err := pool.QueryRow(ctx, `
UPDATE auth_sessions
SET last_seen_at = $2, expires_at = $3
WHERE id = $1
RETURNING id, user_id, token_hash, expires_at, created_at, last_seen_at`,
		session.ID, session.LastSeenAt, session.ExpiresAt).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt); err != nil {
		return Session{}, mapDBError(err)
	}
	session.User, err = r.GetUserAuthByID(ctx, session.UserID)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (r *PostgresRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r *PostgresRepository) RotatePassword(ctx context.Context, userID string, passwordHash string) error {
	if _, err := r.pool(); err != nil {
		return err
	}
	tx, err := r.db.Raw().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackAuthTx(ctx, tx)
	tag, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return mapDBError(err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("auth resource not found")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id = $1`, userID); err != nil {
		return mapDBError(err)
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	pool, err := r.pool()
	if err != nil {
		return 0, err
	}
	tag, err := pool.Exec(ctx, `DELETE FROM auth_sessions WHERE expires_at <= $1`, before)
	if err != nil {
		return 0, mapDBError(err)
	}
	return tag.RowsAffected(), nil
}

func rollbackAuthTx(ctx context.Context, tx pgx.Tx) {
	if tx != nil {
		_ = tx.Rollback(ctx)
	}
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundError("auth resource not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return conflictError("auth resource already exists")
		case "23503":
			return validationError("referenced resource does not exist")
		}
	}
	return fmt.Errorf("auth postgres repository: %w", err)
}
