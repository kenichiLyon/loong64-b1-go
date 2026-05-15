package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kenichiLyon/loong64-b1-go/internal/authn/authnpg"
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
	authnpg.DBTX
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
	row, err := authnpg.New(pool).GetUserAuthByUsername(ctx, username)
	if err != nil {
		return UserAuth{}, mapDBError(err)
	}
	return loadUserAuthRoles(ctx, authnpg.New(pool), UserAuth{
		ID:           row.ID,
		Username:     row.Username,
		DisplayName:  row.DisplayName,
		Status:       row.Status,
		PasswordHash: row.PasswordHash,
	})
}

func (r *PostgresRepository) GetUserAuthByID(ctx context.Context, id string) (UserAuth, error) {
	pool, err := r.pool()
	if err != nil {
		return UserAuth{}, err
	}
	row, err := authnpg.New(pool).GetUserAuthByID(ctx, id)
	if err != nil {
		return UserAuth{}, mapDBError(err)
	}
	return loadUserAuthRoles(ctx, authnpg.New(pool), UserAuth{
		ID:           row.ID,
		Username:     row.Username,
		DisplayName:  row.DisplayName,
		Status:       row.Status,
		PasswordHash: row.PasswordHash,
	})
}

func loadUserAuthRoles(ctx context.Context, queries *authnpg.Queries, user UserAuth) (UserAuth, error) {
	roles, err := queries.ListUserRoles(ctx, user.ID)
	if err != nil {
		return UserAuth{}, mapDBError(err)
	}
	for _, role := range roles {
		user.Roles = append(user.Roles, teaching.Role(role))
	}
	return user, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session Session) (Session, error) {
	pool, err := r.pool()
	if err != nil {
		return Session{}, err
	}
	created, err := authnpg.New(pool).CreateSession(ctx, authnpg.CreateSessionParams{
		ID:         session.ID,
		UserID:     session.UserID,
		TokenHash:  session.TokenHash,
		ExpiresAt:  pgTimestamptz(session.ExpiresAt),
		LastSeenAt: pgTimestamptz(session.LastSeenAt),
	})
	if err != nil {
		return Session{}, mapDBError(err)
	}
	session = sessionFromPG(created)
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
	row, err := authnpg.New(pool).GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return Session{}, mapDBError(err)
	}
	session := sessionFromPG(row)
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
	row, err := authnpg.New(pool).TouchSession(ctx, authnpg.TouchSessionParams{
		ID:         session.ID,
		LastSeenAt: pgTimestamptz(session.LastSeenAt),
		ExpiresAt:  pgTimestamptz(session.ExpiresAt),
	})
	if err != nil {
		return Session{}, mapDBError(err)
	}
	session = sessionFromPG(row)
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
	if err := authnpg.New(pool).DeleteSessionByTokenHash(ctx, tokenHash); err != nil {
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
	queries := authnpg.New(tx)
	rowsAffected, err := queries.RotateUserPassword(ctx, authnpg.RotateUserPasswordParams{ID: userID, PasswordHash: passwordHash})
	if err != nil {
		return mapDBError(err)
	}
	if rowsAffected == 0 {
		return notFoundError("auth resource not found")
	}
	if err := queries.DeleteSessionsByUserID(ctx, userID); err != nil {
		return mapDBError(err)
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	pool, err := r.pool()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := authnpg.New(pool).DeleteExpiredSessions(ctx, pgTimestamptz(before))
	if err != nil {
		return 0, mapDBError(err)
	}
	return rowsAffected, nil
}

func rollbackAuthTx(ctx context.Context, tx pgx.Tx) {
	if tx != nil {
		_ = tx.Rollback(ctx)
	}
}

func pgTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func sessionFromPG(row authnpg.AuthSession) Session {
	return Session{
		ID:         row.ID,
		UserID:     row.UserID,
		TokenHash:  row.TokenHash,
		ExpiresAt:  row.ExpiresAt.Time,
		CreatedAt:  row.CreatedAt.Time,
		LastSeenAt: row.LastSeenAt.Time,
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
