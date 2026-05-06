#!/usr/bin/env sh
set -eu

APP_HOME="${APP_HOME:-/opt/loong64-b1-go}"
APP_CONFIG="${APP_CONFIG:-/etc/loong64-b1-go}"
APP_STATE="${APP_STATE:-/var/lib/loong64-b1-go}"
APP_WEB="${APP_WEB:-$APP_HOME/web}"
APP_ENV_FILE="${APP_ENV_FILE:-$APP_CONFIG/loong64-b1-go.env}"

fail() {
  echo "preflight failed: $1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

need_file() {
  [ -f "$1" ] || fail "missing file: $1"
}

need_dir() {
  [ -d "$1" ] || fail "missing directory: $1"
}

need_cmd systemctl
need_cmd curl
need_cmd sha256sum
need_cmd tar

need_dir "$APP_HOME"
need_dir "$APP_HOME/bin"
need_dir "$APP_STATE"
need_dir "$APP_STATE/storage"
need_dir "$APP_WEB"
need_file "$APP_ENV_FILE"
need_file "$APP_HOME/bin/loong64-b1-go-linux-loong64"
need_file "$APP_HOME/bin/loong64-b1-migrate-linux-loong64"
need_file "$APP_WEB/index.html"

grep -q '^DATABASE_URL=' "$APP_ENV_FILE" || fail "DATABASE_URL is missing in $APP_ENV_FILE"
grep -q '^LLM_BASE_URL=' "$APP_ENV_FILE" || fail "LLM_BASE_URL is missing in $APP_ENV_FILE"
grep -q '^LLM_MODEL=' "$APP_ENV_FILE" || fail "LLM_MODEL is missing in $APP_ENV_FILE"

echo "preflight passed"
