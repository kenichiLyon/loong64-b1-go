package authn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type SQLiteRepository struct {
	db *database.Pool
}

func NewSQLiteRepository(db *database.Pool) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) sqlDB() (*sql.DB, error) {
	if r == nil || r.db == nil || r.db.SQLDB() == nil {
		return nil, unavailableError("auth sqlite repository is not configured", nil)
	}
	return r.db.SQLDB(), nil
}

func (r *SQLiteRepository) GetUserAuthByUsername(ctx context.Context, username string) (UserAuth, error) {
	db, err := r.sqlDB()
	if err != nil {
		return UserAuth{}, err
	}
	var user UserAuth
	if err := db.QueryRowContext(ctx, `
SELECT id, username, display_name, status, COALESCE(password_hash, '') AS password_hash
FROM users WHERE lower(username)=lower(?)`, username).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Status, &user.PasswordHash); err != nil {
		return UserAuth{}, sqliteMapError(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT role FROM user_roles WHERE user_id = ? ORDER BY role`, user.ID)
	if err != nil {
		return UserAuth{}, sqliteMapError(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return UserAuth{}, sqliteMapError(err)
		}
		user.Roles = append(user.Roles, teaching.Role(role))
	}
	return user, rows.Err()
}

func (r *SQLiteRepository) GetUserAuthByID(ctx context.Context, id string) (UserAuth, error) {
	db, err := r.sqlDB()
	if err != nil {
		return UserAuth{}, err
	}
	var user UserAuth
	if err := db.QueryRowContext(ctx, `
SELECT id, username, display_name, status, COALESCE(password_hash, '') AS password_hash
FROM users WHERE id=?`, id).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Status, &user.PasswordHash); err != nil {
		return UserAuth{}, sqliteMapError(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT role FROM user_roles WHERE user_id = ? ORDER BY role`, user.ID)
	if err != nil {
		return UserAuth{}, sqliteMapError(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return UserAuth{}, sqliteMapError(err)
		}
		user.Roles = append(user.Roles, teaching.Role(role))
	}
	return user, rows.Err()
}

func (r *SQLiteRepository) CreateSession(ctx context.Context, session Session) (Session, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Session{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO auth_sessions (id, user_id, token_hash, expires_at, last_seen_at)
VALUES (?, ?, ?, ?, ?)`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.LastSeenAt); err != nil {
		return Session{}, sqliteMapError(err)
	}
	return r.GetSessionByTokenHash(ctx, session.TokenHash)
}

func (r *SQLiteRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := db.QueryRowContext(ctx, `
SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at
FROM auth_sessions WHERE token_hash = ?`, tokenHash).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastSeenAt); err != nil {
		return Session{}, sqliteMapError(err)
	}
	session.User, err = r.GetUserAuthByID(ctx, session.UserID)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (r *SQLiteRepository) TouchSession(ctx context.Context, session Session) (Session, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Session{}, err
	}
	if _, err := db.ExecContext(ctx, `UPDATE auth_sessions SET last_seen_at = ? WHERE id = ?`, session.LastSeenAt, session.ID); err != nil {
		return Session{}, sqliteMapError(err)
	}
	return r.GetSessionByTokenHash(ctx, session.TokenHash)
}

func (r *SQLiteRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	db, err := r.sqlDB()
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return sqliteMapError(err)
	}
	return nil
}

func (r *SQLiteRepository) RotatePassword(ctx context.Context, userID string, passwordHash string) error {
	db, err := r.sqlDB()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackSQLiteAuthTx(ctx, tx)
	result, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, passwordHash, userID)
	if err != nil {
		return sqliteMapError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return sqliteMapError(err)
	}
	if rows == 0 {
		return notFoundError("auth resource not found")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id = ?`, userID); err != nil {
		return sqliteMapError(err)
	}
	return tx.Commit()
}

func (r *SQLiteRepository) DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	db, err := r.sqlDB()
	if err != nil {
		return 0, err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE expires_at <= ?`, before)
	if err != nil {
		return 0, sqliteMapError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, sqliteMapError(err)
	}
	return rows, nil
}

func rollbackSQLiteAuthTx(_ context.Context, tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func sqliteMapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFoundError("auth resource not found")
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "unique constraint failed"):
		return conflictError("auth resource already exists")
	case strings.Contains(text, "foreign key constraint failed"):
		return validationError("referenced resource does not exist")
	}
	return fmt.Errorf("auth sqlite repository: %w", err)
}
