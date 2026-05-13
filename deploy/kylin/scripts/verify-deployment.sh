#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

check_json() {
  endpoint="$1"
  expected_fragment="$2"
  body="$(curl -fsS "$BASE_URL$endpoint")"
  echo "$body" | grep '"status":"ok"' >/dev/null
  echo "$body" | grep "$expected_fragment" >/dev/null
}

echo "Checking live health"
check_json "/health/live" '"service":"loong64-b1-go"'

echo "Checking ready health"
check_json "/health/ready" '"service":"loong64-b1-go"'

echo "Checking root metadata"
root_body="$(curl -fsS "$BASE_URL/")"
echo "$root_body" | grep '"service":"loong64-b1-go"' >/dev/null
echo "$root_body" | grep '"live":"/health/live"' >/dev/null
echo "$root_body" | grep '"ready":"/health/ready"' >/dev/null

echo "Deployment verification passed."
