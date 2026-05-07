# 容器次级交付

本文件描述 `loong64-b1-go` 的容器化次级交付。它的定位是开发、演示、CI 冒烟和非银河麒麟环境快速试跑，不替代主交付的 `LoongArch + 银河麒麟 + systemd` 部署路线。

## 定位

- 主交付：`systemd` 非容器部署
- 次级交付：`Docker / Podman` 本地开发与演示
- 默认模式：内嵌前端 + SQLite 卷持久化
- 可选模式：`compose` 中启用 PostgreSQL profile

## 文件

- `Containerfile`：多阶段构建，产出内嵌前端的单二进制镜像
- `.dockerignore`：裁剪构建上下文
- `compose.yaml`：默认 SQLite 服务与可选 PostgreSQL profile
- `.github/workflows/container-smoke.yml`：在 GitHub Actions 上执行镜像构建与默认 SQLite 启动冒烟

镜像内已包含 `/migrations`，并设置 `MIGRATIONS_DIR=/migrations`，因此默认 SQLite 容器会在启动时自动完成迁移。

## 构建

Docker：

```bash
docker build -t loong64-b1-go:dev -f Containerfile .
```

Podman：

```bash
podman build -t loong64-b1-go:dev -f Containerfile .
```

## 运行

默认 SQLite：

```bash
docker compose up --build app
```

或：

```bash
podman compose up --build app
```

服务默认监听 `8080`，数据库与运行配置都保存在容器卷 `app-data`。

## PostgreSQL Profile

如果需要联调 PostgreSQL：

```bash
docker compose --profile postgres up --build postgres app-postgres
```

此模式下：

- `postgres` 监听宿主机 `5432`
- `app-postgres` 监听宿主机 `8081`
- 应用通过 `DATABASE_URL` 连接容器内 PostgreSQL
- `AUTO_MIGRATE=true`，容器启动时自动迁移

## 环境变量

默认 SQLite 容器环境：

```env
DB_DRIVER=sqlite
SQLITE_PATH=/var/lib/loong64-b1-go/data/loong64-b1-go.db
AUTO_MIGRATE=true
STORAGE_ROOT=/var/lib/loong64-b1-go/storage
RUNTIME_CONFIG_PATH=/var/lib/loong64-b1-go/config/runtime.json
```

PostgreSQL 容器环境：

```env
DB_DRIVER=postgres
DATABASE_URL=postgres://loong64_b1:loong64_b1_dev@postgres:5432/loong64_b1?sslmode=disable
AUTO_MIGRATE=true
```

## 限制

- 当前环境没有安装 `docker` / `podman` / `buildah`，本轮未执行真实镜像构建，只做了容器文件静态校验与 `compose` YAML 结构校验。
- 容器交付默认面向 `linux/amd64` / 常见开发环境；LoongArch 正式环境仍以 systemd 主方案为准。
- 容器模式下的生产级监控、日志采集和镜像发布策略仍未接入。

## CI 冒烟

仓库已增加 `Container Smoke` 工作流，默认执行：

1. `docker buildx build --file Containerfile --load .`
2. 启动默认 SQLite 容器
3. 轮询 `/health/ready`
4. 访问 `/api/v1/bootstrap/status`

这条流水线的目标是保证次级交付资产在 PR 中不失效；它不是正式发布镜像的发布链。
