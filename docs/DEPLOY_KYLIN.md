# 银河麒麟 + LoongArch 部署骨架

本文件记录 systemd 非容器部署的首版骨架。目标环境为 LoongArch 架构 + 银河麒麟高级服务器版。

## 1. 发布产物

CD 发布后，从 GitHub Release 下载：

- `loong64-b1-go-linux-loong64`
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

## 5. 部署前端静态资源

```bash
sudo install -o loong64b1 -g loong64b1 -d -m 0755 /opt/loong64-b1-go/web
sudo tar -xzf loong64-b1-go-web.tar.gz -C /opt/loong64-b1-go/web
sudo chown -R loong64b1:loong64b1 /opt/loong64-b1-go/web
```

当前 Go API 服务不内置静态资源托管，建议在银河麒麟上使用 Nginx、Apache 或学校统一 Web 网关托管 `/opt/loong64-b1-go/web`，并把 `/api` 与 `/health` 反向代理到 `http://127.0.0.1:8080`。

编辑 `/etc/loong64-b1-go/loong64-b1-go.env`，至少修改：

- `DATABASE_URL`
- `LLM_BASE_URL`
- `LLM_MODEL`
- `LLM_API_KEY`，如使用需要鉴权的模型网关

## 6. 数据库迁移与启动

```bash
sudo systemctl start loong64-b1-migrate.service
sudo systemctl enable --now loong64-b1-go.service
sudo systemctl status loong64-b1-go.service
```

## 7. 冒烟测试

```bash
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/smoke-test.sh
```

或手动执行：

```bash
curl -fsS http://127.0.0.1:8080/health/live
curl -fsS http://127.0.0.1:8080/health/ready
```

`ready` 必须覆盖 PostgreSQL 和本地对象存储。

## 8. LoongArch 记录

首次部署必须把以下信息追加到 `docs/LOONGARCH_COMPATIBILITY.md`：

```bash
uname -m
uname -a
cat /etc/os-release
go version || true
psql --version
systemctl status loong64-b1-go.service --no-pager
```
