# 本地 PostgreSQL 开发脚本

本项目默认使用 PostgreSQL。开发环境可以是 Windows amd64 或 Linux amd64。

## Windows PowerShell

要求已安装 PostgreSQL 客户端并可执行 `psql`。

```powershell
$env:POSTGRES_SUPERUSER_URL='postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable'
.\scripts\dev\postgres-init.ps1
$env:DATABASE_URL='postgres://loong64_b1:loong64_b1_dev@127.0.0.1:5432/loong64_b1?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/server
```

也可以直接运行：

```powershell
.\scripts\dev\run-local.ps1
```

## Linux / macOS Shell

```bash
POSTGRES_SUPERUSER_URL='postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable' \
  sh scripts/dev/postgres-init.sh

export DATABASE_URL='postgres://loong64_b1:loong64_b1_dev@127.0.0.1:5432/loong64_b1?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/server
```

## 注意事项

- 脚本只创建本地开发库和开发用户，不用于生产环境。
- 生产环境按 `docs/DEPLOY_KYLIN.md` 配置独立密码和最小权限账号。
- 不要把真实 `DATABASE_URL` 或密码写入仓库。
