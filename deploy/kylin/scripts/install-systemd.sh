#!/usr/bin/env sh
set -eu

APP_USER="${APP_USER:-loong64b1}"
APP_GROUP="${APP_GROUP:-loong64b1}"
APP_HOME="${APP_HOME:-/opt/loong64-b1-go}"
APP_STATE="${APP_STATE:-/var/lib/loong64-b1-go}"
APP_LOG="${APP_LOG:-/var/log/loong64-b1-go}"
APP_CONFIG="${APP_CONFIG:-/etc/loong64-b1-go}"

if [ "$(id -u)" -ne 0 ]; then
  echo "install-systemd.sh must run as root" >&2
  exit 1
fi

if ! getent group "$APP_GROUP" >/dev/null 2>&1; then
  groupadd --system "$APP_GROUP"
fi

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --gid "$APP_GROUP" --home-dir "$APP_HOME" --shell /sbin/nologin "$APP_USER"
fi

install -d -o "$APP_USER" -g "$APP_GROUP" -m 0750 "$APP_HOME" "$APP_HOME/bin" "$APP_STATE" "$APP_STATE/storage" "$APP_LOG"
install -d -o root -g "$APP_GROUP" -m 0750 "$APP_CONFIG"

if [ ! -f "$APP_CONFIG/loong64-b1-go.env" ]; then
  install -o root -g "$APP_GROUP" -m 0640 deploy/kylin/env/loong64-b1-go.env.example "$APP_CONFIG/loong64-b1-go.env"
  echo "Created $APP_CONFIG/loong64-b1-go.env; edit DATABASE_URL and LLM settings before starting."
fi

install -o root -g root -m 0644 deploy/kylin/systemd/loong64-b1-go.service /etc/systemd/system/loong64-b1-go.service
install -o root -g root -m 0644 deploy/kylin/systemd/loong64-b1-migrate.service /etc/systemd/system/loong64-b1-migrate.service

systemctl daemon-reload
echo "Systemd units installed. Copy release binaries into $APP_HOME/bin, then run:"
echo "  systemctl start loong64-b1-migrate.service"
echo "  systemctl enable --now loong64-b1-go.service"
