-- name: GetUserAuthByUsername :one
SELECT id, username, display_name, status, COALESCE(password_hash, '')::text AS password_hash
FROM users
WHERE lower(username) = lower($1);

-- name: GetUserAuthByID :one
SELECT id, username, display_name, status, COALESCE(password_hash, '')::text AS password_hash
FROM users
WHERE id = $1;

-- name: ListUserRoles :many
SELECT role
FROM user_roles
WHERE user_id = $1
ORDER BY role;

-- name: CreateSession :one
INSERT INTO auth_sessions (id, user_id, token_hash, expires_at, last_seen_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, expires_at, created_at, last_seen_at;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at
FROM auth_sessions
WHERE token_hash = $1;

-- name: TouchSession :one
UPDATE auth_sessions
SET last_seen_at = $2, expires_at = $3
WHERE id = $1
RETURNING id, user_id, token_hash, expires_at, created_at, last_seen_at;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM auth_sessions
WHERE token_hash = $1;

-- name: RotateUserPassword :execrows
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM auth_sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM auth_sessions
WHERE expires_at <= $1;
