#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
EXPECTED_STATUS="${EXPECTED_STATUS:-200}"

check_json() {
  endpoint="$1"
  expected="$2"
  body="$(curl -fsS "$BASE_URL$endpoint")"
  echo "$body" | grep '"status":"ok"' >/dev/null
  echo "$body" | grep '"service":"loong64-b1-go"' >/dev/null
  if [ -n "$expected" ]; then
    echo "$body" | grep "$expected" >/dev/null
  fi
}

echo "Checking live health"
check_json "/health/live" '"status":"ok"'

echo "Checking ready health"
check_json "/health/ready" '"status":"ok"'

echo "Checking root metadata"
root_body="$(curl -fsS "$BASE_URL/")"
echo "$root_body" | grep '"service":"loong64-b1-go"' >/dev/null
echo "$root_body" | grep '"live":"/health/live"' >/dev/null
echo "$root_body" | grep '"ready":"/health/ready"' >/dev/null

echo "Deployment verification passed."
