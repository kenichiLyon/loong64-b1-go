# 银河麒麟 + LoongArch 部署与验证

本文件记录 systemd 非容器部署与 Stage 7 验证流程。目标环境为 LoongArch 架构 + 银河麒麟高级服务器版。

## 1. 发布产物

CD 发布后，从 GitHub Release 下载：

- `loong64-b1-go-linux-loong64`
- `loong64-b1-go-linux-loong64-full`
- `loong64-b1-migrate-linux-loong64`
- `loong64-b1-go-web.tar.gz`
- `SHA256SUMS`

建议先校验：

```bash
sha256sum -c SHA256SUMS
```

## 2. 安装目录

默认目录：

```text
/opt/loong64-b1-go/bin
/opt/loong64-b1-go/web
/etc/loong64-b1-go/loong64-b1-go.env
/etc/loong64-b1-go/runtime.json
/var/lib/loong64-b1-go/storage
/var/log/loong64-b1-go
```

## 3. 安装 systemd 单元

在仓库根目录执行：

```bash
sudo sh deploy/kylin/scripts/install-systemd.sh
```

脚本会创建：

- 系统用户：`loong64b1`
- 服务单元：`loong64-b1-go.service`
- 迁移单元：`loong64-b1-migrate.service`
- 配置模板：`/etc/loong64-b1-go/loong64-b1-go.env`

## 4. 部署二进制

```bash
sudo install -o loong64b1 -g loong64b1 -m 0755 loong64-b1-go-linux-loong64 /opt/loong64-b1-go/bin/
sudo install -o loong64b1 -g loong64b1 -m 0755 loong64-b1-migrate-linux-loong64 /opt/loong64-b1-go/bin/
```

如果希望仅靠一个服务二进制同时提供 API 和前端页面，使用：

```bash
sudo install -o loong64b1 -g loong64b1 -m 0755 loong64-b1-go-linux-loong64-full /opt/loong64-b1-go/bin/loong64-b1-go-linux-loong64
```

## 5. 部署前端静态资源

```bash
sudo install -o loong64b1 -g loong64b1 -d -m 0755 /opt/loong64-b1-go/web
sudo tar -xzf loong64-b1-go-web.tar.gz -C /opt/loong64-b1-go/web
sudo chown -R loong64b1:loong64b1 /opt/loong64-b1-go/web
```

使用 `loong64-b1-go-linux-loong64-full` 时，可以跳过本节，直接由服务二进制托管 PC Web 页面。使用纯后端二进制时，仍建议在银河麒麟上使用 Nginx、Apache 或学校统一 Web 网关托管 `/opt/loong64-b1-go/web`，并把 `/api` 与 `/health` 反向代理到 `http://127.0.0.1:8080`。Nginx 示例见 `deploy/kylin/nginx/loong64-b1-go.conf.example`。

编辑 `/etc/loong64-b1-go/loong64-b1-go.env`，至少修改：

- `DB_DRIVER`
- `SESSION_SECURE_COOKIE`
- `LLM_BASE_URL`
- `LLM_MODEL`
- `LLM_API_KEY`，如使用需要鉴权的模型网关

SQLite 默认示例：

```env
DB_DRIVER=sqlite
SQLITE_PATH=/var/lib/loong64-b1-go/data/loong64-b1-go.db
```

PostgreSQL 示例：

```env
DB_DRIVER=postgres
DATABASE_URL=postgres://loong64_b1:CHANGE_ME@127.0.0.1:5432/loong64_b1?sslmode=disable
```

认证 cookie 建议：

```env
SESSION_COOKIE_NAME=loong64_b1_session
SESSION_TTL=168h
SESSION_SECURE_COOKIE=true
```

本地模型示例：

```env
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
LLM_API_KEY=
```

云端或校内网关示例：

```env
LLM_BASE_URL=https://llm-gateway.example.edu/v1
LLM_MODEL=gpt-compatible-model
LLM_API_KEY=REDACTED
```

## 6. 数据库迁移与启动

```bash
sudo systemctl start loong64-b1-migrate.service
sudo systemctl enable --now loong64-b1-go.service
sudo systemctl status loong64-b1-go.service
sh deploy/kylin/scripts/preflight-check.sh
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/verify-deployment.sh
```

## 7. 冒烟测试

```bash
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/smoke-test.sh
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/verify-deployment.sh
```

或手动执行：

```bash
curl -fsS http://127.0.0.1:8080/health/live
curl -fsS http://127.0.0.1:8080/health/ready
```

`ready` 必须覆盖当前数据库驱动和本地对象存储。

## 8. 备份与恢复

数据库备份：

```bash
pg_dump --format=custom --file /var/backups/loong64-b1-go/db.dump "$DATABASE_URL"
tar -czf /var/backups/loong64-b1-go/storage.tar.gz /var/lib/loong64-b1-go/storage
```

数据库恢复：

```bash
createdb loong64_b1_restore
pg_restore --clean --if-exists --no-owner --dbname loong64_b1_restore /var/backups/loong64-b1-go/db.dump
tar -xzf /var/backups/loong64-b1-go/storage.tar.gz -C /
```

## 9. LoongArch 记录

首次部署必须把以下信息追加到 `docs/LOONGARCH_COMPATIBILITY.md`：

```bash
sh deploy/kylin/scripts/collect-env.sh /tmp/loong64-b1-go-stage7.txt
```

完整验收清单见 `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`。
