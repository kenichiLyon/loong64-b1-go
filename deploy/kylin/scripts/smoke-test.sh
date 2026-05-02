#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

echo "Checking $BASE_URL/health/live"
curl -fsS "$BASE_URL/health/live" >/dev/null

echo "Checking $BASE_URL/health/ready"
curl -fsS "$BASE_URL/health/ready" >/dev/null

echo "Smoke test passed."
