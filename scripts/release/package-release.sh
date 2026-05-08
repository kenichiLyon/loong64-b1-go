#!/usr/bin/env bash
set -euo pipefail

RAW_DIR="${1:-dist/raw}"
OUT_DIR="${2:-dist/release}"
STAGE_DIR="${OUT_DIR}/_stage"

FULL_BUNDLE="loong64-b1-go-full-linux-loong64.tar.gz"
BACKEND_BUNDLE="loong64-b1-go-backend-linux-loong64.tar.gz"
FRONTEND_BUNDLE="loong64-b1-go-frontend.tar.gz"

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

require_file "${RAW_DIR}/loong64-b1-go-linux-loong64"
require_file "${RAW_DIR}/loong64-b1-go-linux-loong64-full"
require_file "${RAW_DIR}/loong64-b1-migrate-linux-loong64"
require_dir "web/dist"
require_dir "migrations"
require_dir "deploy/kylin"

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}" "${STAGE_DIR}/full" "${STAGE_DIR}/backend" "${STAGE_DIR}/frontend"

mkdir -p "${STAGE_DIR}/full/bin" "${STAGE_DIR}/full/config" "${STAGE_DIR}/full/docs" "${STAGE_DIR}/full/deploy"
cp "${RAW_DIR}/loong64-b1-go-linux-loong64-full" "${STAGE_DIR}/full/bin/"
cp "deploy/kylin/env/loong64-b1-go.env.example" "${STAGE_DIR}/full/config/"
cp -R "migrations" "${STAGE_DIR}/full/"
cp -R "deploy/kylin/scripts" "${STAGE_DIR}/full/deploy/"
cp -R "deploy/kylin/systemd" "${STAGE_DIR}/full/deploy/"
cp "docs/SINGLE_BINARY_RUNTIME.md" "docs/DEPLOY_KYLIN.md" "docs/MVP_DELIVERY.md" "${STAGE_DIR}/full/docs/"
cat > "${STAGE_DIR}/full/README.txt" <<'EOF'
Primary MVP delivery bundle.

Contents:
- bin/loong64-b1-go-linux-loong64-full
- migrations/
- deploy/kylin/scripts/
- deploy/kylin/systemd/
- config/loong64-b1-go.env.example
- docs/SINGLE_BINARY_RUNTIME.md
- docs/DEPLOY_KYLIN.md
- docs/MVP_DELIVERY.md

Use this bundle by default unless you explicitly need split frontend/backend deployment.
EOF

mkdir -p "${STAGE_DIR}/backend/bin" "${STAGE_DIR}/backend/config" "${STAGE_DIR}/backend/docs" "${STAGE_DIR}/backend/deploy"
cp "${RAW_DIR}/loong64-b1-go-linux-loong64" "${STAGE_DIR}/backend/bin/"
cp "${RAW_DIR}/loong64-b1-migrate-linux-loong64" "${STAGE_DIR}/backend/bin/"
cp "deploy/kylin/env/loong64-b1-go.env.example" "${STAGE_DIR}/backend/config/"
cp -R "migrations" "${STAGE_DIR}/backend/"
cp -R "deploy/kylin/scripts" "${STAGE_DIR}/backend/deploy/"
cp -R "deploy/kylin/systemd" "${STAGE_DIR}/backend/deploy/"
cp -R "deploy/kylin/nginx" "${STAGE_DIR}/backend/deploy/"
cp "docs/DEPLOY_KYLIN.md" "docs/MVP_DELIVERY.md" "${STAGE_DIR}/backend/docs/"
cat > "${STAGE_DIR}/backend/README.txt" <<'EOF'
Split deployment backend bundle.

Contents:
- bin/loong64-b1-go-linux-loong64
- bin/loong64-b1-migrate-linux-loong64
- migrations/
- deploy/kylin/scripts/
- deploy/kylin/systemd/
- deploy/kylin/nginx/
- config/loong64-b1-go.env.example
- docs/DEPLOY_KYLIN.md
- docs/MVP_DELIVERY.md

Use this only when frontend and backend must be deployed separately.
EOF

mkdir -p "${STAGE_DIR}/frontend/web" "${STAGE_DIR}/frontend/deploy" "${STAGE_DIR}/frontend/docs"
cp -R web/dist/. "${STAGE_DIR}/frontend/web/"
cp -R "deploy/kylin/nginx" "${STAGE_DIR}/frontend/deploy/"
cp "docs/DEPLOY_KYLIN.md" "docs/MVP_DELIVERY.md" "${STAGE_DIR}/frontend/docs/"
cat > "${STAGE_DIR}/frontend/README.txt" <<'EOF'
Split deployment frontend bundle.

Contents:
- web/ (Vite production static files)
- deploy/kylin/nginx/
- docs/DEPLOY_KYLIN.md
- docs/MVP_DELIVERY.md

Use this only when frontend must be hosted separately from the Go service.
EOF

tar -C "${STAGE_DIR}/full" -czf "${OUT_DIR}/${FULL_BUNDLE}" .
tar -C "${STAGE_DIR}/backend" -czf "${OUT_DIR}/${BACKEND_BUNDLE}" .
tar -C "${STAGE_DIR}/frontend" -czf "${OUT_DIR}/${FRONTEND_BUNDLE}" .

(cd "${OUT_DIR}" && sha256sum "${FULL_BUNDLE}" "${BACKEND_BUNDLE}" "${FRONTEND_BUNDLE}" > SHA256SUMS)
rm -rf "${STAGE_DIR}"
