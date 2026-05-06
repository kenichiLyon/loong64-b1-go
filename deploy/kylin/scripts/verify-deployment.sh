#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
WEB_ROOT="${WEB_ROOT:-/opt/loong64-b1-go/web}"
EXPECTED_ARCH="${EXPECTED_ARCH:-loongarch64}"

echo "Running preflight checks"
sh "$(dirname "$0")/preflight-check.sh"

ACTUAL_ARCH="$(uname -m)"
if [ "$ACTUAL_ARCH" != "$EXPECTED_ARCH" ]; then
  echo "warning: expected arch $EXPECTED_ARCH but got $ACTUAL_ARCH" >&2
fi

[ -f "$WEB_ROOT/index.html" ] || {
  echo "verification failed: missing $WEB_ROOT/index.html" >&2
  exit 1
}

echo "Checking service active state"
systemctl is-active --quiet loong64-b1-go.service

echo "Checking $BASE_URL/health/live"
curl -fsS "$BASE_URL/health/live" >/dev/null

echo "Checking $BASE_URL/health/ready"
curl -fsS "$BASE_URL/health/ready" >/dev/null

echo "Deployment verification passed."
