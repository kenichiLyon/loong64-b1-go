$ErrorActionPreference = 'Stop'

$dbName = if ($env:DB_NAME) { $env:DB_NAME } else { 'loong64_b1' }
$dbUser = if ($env:DB_USER) { $env:DB_USER } else { 'loong64_b1' }
$dbPassword = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { 'loong64_b1_dev' }
$superuserUrl = if ($env:POSTGRES_SUPERUSER_URL) { $env:POSTGRES_SUPERUSER_URL } else { 'postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable' }

if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
    throw 'psql is required. Install PostgreSQL client first.'
}

$sql = @"
DO `$`$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$dbUser') THEN
    EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', '$dbUser', '$dbPassword');
  END IF;
END
`$`$;

SELECT format('CREATE DATABASE %I OWNER %I', '$dbName', '$dbUser')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '$dbName')\gexec
"@

$sql | psql $superuserUrl -v ON_ERROR_STOP=1
Write-Output "Database ready: postgres://$dbUser:$dbPassword@127.0.0.1:5432/$dbName?sslmode=disable"
