#!/usr/bin/env sh
set -eu

OUT_FILE="${1:-/tmp/loong64-b1-go-stage7.txt}"
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

collect() {
  {
    echo "date=$(date -Iseconds)"
    echo "uname_m=$(uname -m)"
    echo "uname_a=$(uname -a)"
    echo "--- os-release ---"
    cat /etc/os-release 2>/dev/null || true
    echo "--- go version ---"
    go version 2>/dev/null || true
    echo "--- psql version ---"
    psql --version 2>/dev/null || true
    echo "--- systemd status ---"
    systemctl status loong64-b1-go.service --no-pager 2>/dev/null || true
    echo "--- health live ---"
    curl -fsS "$BASE_URL/health/live" 2>/dev/null || true
    echo
    echo "--- health ready ---"
    curl -fsS "$BASE_URL/health/ready" 2>/dev/null || true
    echo
  } > "$OUT_FILE"
}

collect

echo "Environment snapshot written to $OUT_FILE"
