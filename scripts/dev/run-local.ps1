$ErrorActionPreference = 'Stop'

if (-not $env:DATABASE_URL) {
    $env:DATABASE_URL = 'postgres://loong64_b1:loong64_b1_dev@127.0.0.1:5432/loong64_b1?sslmode=disable'
}
if (-not $env:STORAGE_ROOT) {
    $env:STORAGE_ROOT = './storage'
}

go run ./cmd/migrate up
go run ./cmd/server
