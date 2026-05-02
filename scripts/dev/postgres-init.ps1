$ErrorActionPreference = 'Stop'

$dbName = if ($env:DB_NAME) { $env:DB_NAME } else { 'loong64_b1' }
$dbUser = if ($env:DB_USER) { $env:DB_USER } else { 'loong64_b1' }
$dbPassword = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { 'loong64_b1_dev' }
$superuserUrl = if ($env:POSTGRES_SUPERUSER_URL) { $env:POSTGRES_SUPERUSER_URL } else { 'postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable' }

if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
    throw 'psql is required. Install PostgreSQL client first.'
}

$sql = @'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'db_user', :'db_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'db_user')\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'db_name', :'db_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db_name')\gexec
'@

$sql | psql $superuserUrl -v ON_ERROR_STOP=1 -v "db_name=$dbName" -v "db_user=$dbUser" -v "db_password=$dbPassword"
Write-Output "Database ready: postgres://${dbUser}:${dbPassword}@127.0.0.1:5432/${dbName}?sslmode=disable"
