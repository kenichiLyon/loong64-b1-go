#!/usr/bin/env bash
set -euo pipefail

RAW_DIR="${1:-dist/raw}"
OUT_DIR="${2:-dist/release}"
STAGE_DIR="${OUT_DIR}/_stage"

FULL_BUNDLE="loong64-b1-go-full-linux-loong64.tar.gz"

require_file() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
}

require_dir() {
  local dir="$1"
  if [[ ! -d "$dir" ]]; then
    echo "missing required directory: $dir" >&2
    exit 1
  fi
}

require_file "${RAW_DIR}/loong64-b1-go-linux-loong64-full"
require_file "${RAW_DIR}/loong64-b1-upgrade-linux-loong64"
require_dir "migrations"
require_dir "deploy/kylin"
require_dir "python-ai-gateway"

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}" "${STAGE_DIR}/full"

mkdir -p "${STAGE_DIR}/full/bin" "${STAGE_DIR}/full/config" "${STAGE_DIR}/full/docs" "${STAGE_DIR}/full/deploy" "${STAGE_DIR}/full/python-ai-gateway"
cp "${RAW_DIR}/loong64-b1-go-linux-loong64-full" "${STAGE_DIR}/full/bin/loong64-b1-go-linux-loong64"
cp "${RAW_DIR}/loong64-b1-upgrade-linux-loong64" "${STAGE_DIR}/full/bin/"
cp "deploy/kylin/env/loong64-b1-go.env.example" "${STAGE_DIR}/full/config/"
cp -R "migrations" "${STAGE_DIR}/full/"
cp -R "deploy/kylin" "${STAGE_DIR}/full/deploy/"
tar -C "python-ai-gateway" \
  --exclude=".venv" \
  --exclude="__pycache__" \
  --exclude=".pytest_cache" \
  -cf - . | tar -C "${STAGE_DIR}/full/python-ai-gateway" -xf -
cp "docs/SINGLE_BINARY_RUNTIME.md" "docs/DEPLOY_KYLIN.md" "docs/MVP_DELIVERY.md" "docs/PYTHON_AI_MIDDLEWARE.md" "${STAGE_DIR}/full/docs/"
cat > "${STAGE_DIR}/full/README.txt" <<'EOF'
Primary MVP delivery bundle.

Contents:
- bin/loong64-b1-go-linux-loong64
- bin/loong64-b1-upgrade-linux-loong64
- migrations/
- deploy/kylin/
- python-ai-gateway/
- config/loong64-b1-go.env.example
- docs/SINGLE_BINARY_RUNTIME.md
- docs/DEPLOY_KYLIN.md
- docs/MVP_DELIVERY.md
- docs/PYTHON_AI_MIDDLEWARE.md

The Go service binary embeds web/dist and serves the PC Web UI directly.
EOF

tar -C "${STAGE_DIR}/full" -czf "${OUT_DIR}/${FULL_BUNDLE}" .

(cd "${OUT_DIR}" && sha256sum "${FULL_BUNDLE}" > SHA256SUMS)
rm -rf "${STAGE_DIR}"
