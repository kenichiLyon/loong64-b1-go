#!/usr/bin/env sh
set -eu

APP_HOME="${APP_HOME:-/opt/loong64-b1-go}"
APP_CONFIG="${APP_CONFIG:-/etc/loong64-b1-go}"
APP_STATE="${APP_STATE:-/var/lib/loong64-b1-go}"
APP_LOG="${APP_LOG:-/var/log/loong64-b1-go}"
APP_USER="${APP_USER:-loong64b1}"
APP_GROUP="${APP_GROUP:-loong64b1}"

require_path() {
  target="$1"
  if [ ! -e "$target" ]; then
    echo "missing required path: $target" >&2
    exit 1
  fi
}

require_path "$APP_HOME"
require_path "$APP_HOME/bin"
require_path "$APP_CONFIG"
require_path "$APP_STATE"
require_path "$APP_LOG"
require_path "/etc/systemd/system/loong64-b1-go.service"
require_path "/etc/systemd/system/loong64-b1-upgrade.service"

if ! getent group "$APP_GROUP" >/dev/null 2>&1; then
  echo "missing group: $APP_GROUP" >&2
  exit 1
fi

if ! id "$APP_USER" >/dev/null 2>&1; then
  echo "missing user: $APP_USER" >&2
  exit 1
fi

if [ ! -f "$APP_CONFIG/loong64-b1-go.env" ]; then
  echo "missing env file: $APP_CONFIG/loong64-b1-go.env" >&2
  exit 1
fi

echo "Preflight check passed."
