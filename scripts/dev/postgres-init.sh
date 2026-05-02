#!/usr/bin/env sh
set -eu

DB_NAME="${DB_NAME:-loong64_b1}"
DB_USER="${DB_USER:-loong64_b1}"
DB_PASSWORD="${DB_PASSWORD:-loong64_b1_dev}"
POSTGRES_SUPERUSER_URL="${POSTGRES_SUPERUSER_URL:-postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required. Install PostgreSQL client first." >&2
  exit 1
fi

psql "$POSTGRES_SUPERUSER_URL" -v ON_ERROR_STOP=1 \
  -v db_name="$DB_NAME" \
  -v db_user="$DB_USER" \
  -v db_password="$DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'db_user', :'db_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'db_user')\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'db_name', :'db_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db_name')\gexec
SQL

echo "Database ready: postgres://$DB_USER:$DB_PASSWORD@127.0.0.1:5432/$DB_NAME?sslmode=disable"
