# 银河麒麟 + LoongArch 部署说明

当前正式部署形态不是“只跑一个 Go 服务”，而是：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

也就是：

- `Go 主服务`
- `Python AI Worker`

共同组成正式交付。

Go 主服务二进制默认内嵌 `web/dist`，浏览器直接访问 Go 服务即可使用 PC Web UI。部署链路不再提供前后端拆分发布包。
`python-ai-gateway.service` 是当前 AI Worker 的兼容服务名，后续服务形态升级时应优先按 AI Worker 边界演进，而不是把 AI 能力塞回 Go 主服务。

## 1. 发布产物

优先使用：

- `loong64-b1-go-full-linux-loong64.tar.gz`

部署前建议先校验：

```bash
sha256sum loong64-b1-go-full-linux-loong64.tar.gz
```

## 2. 推荐目录

```text
/opt/loong64-b1-go/bin
/opt/loong64-b1-go/python-ai-gateway
/etc/loong64-b1-go/loong64-b1-go.env
/etc/loong64-b1-go/python-ai-gateway.env
/etc/loong64-b1-go/runtime.json
/var/lib/loong64-b1-go/storage
/var/log/loong64-b1-go
```

## 3. 安装 systemd 资产

在仓库根目录执行：

```bash
sudo sh deploy/kylin/scripts/install-systemd.sh
```

脚本会创建：

- 用户：`loong64b1`
- `loong64-b1-go.service`
- `loong64-b1-upgrade.service`
- `python-ai-gateway.service`
- `/etc/loong64-b1-go/loong64-b1-go.env`
- `/etc/loong64-b1-go/python-ai-gateway.env`

## 4. 部署 Go 主服务

```bash
sudo install -d -m 0755 /opt/loong64-b1-go
sudo tar -xzf loong64-b1-go-full-linux-loong64.tar.gz -C /opt/loong64-b1-go
sudo chown -R loong64b1:loong64b1 /opt/loong64-b1-go
```

## 5. 部署 Python AI Worker

当前实现目录仍是 `python-ai-gateway/`，推荐同步到：

```text
/opt/loong64-b1-go/python-ai-gateway
```

然后在目标机安装依赖：

```bash
cd /opt/loong64-b1-go/python-ai-gateway
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
```

如果目标机没有外网：

- 使用预先下载的 wheel 缓存，或
- 使用校内镜像源

这一点必须留档。

## 6. Go 环境变量

编辑：

```text
/etc/loong64-b1-go/loong64-b1-go.env
```

最小示例：

```env
DB_DRIVER=sqlite
SQLITE_PATH=/var/lib/loong64-b1-go/data/loong64-b1-go.db

AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_API_KEY=
AI_GATEWAY_TIMEOUT=10s
```

如果使用 PostgreSQL：

```env
DB_DRIVER=postgres
DATABASE_URL=postgres://loong64_b1:CHANGE_ME@127.0.0.1:5432/loong64_b1?sslmode=disable
```

## 7. AI Worker 环境变量

编辑：

```text
/etc/loong64-b1-go/python-ai-gateway.env
```

最小示例：

```env
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

如果接校内 / 云端模型网关：

```env
AI_GATEWAY_LLM_BASE_URL=https://llm-gateway.example.edu/v1
AI_GATEWAY_LLM_API_KEY=REDACTED
AI_GATEWAY_LLM_MODEL=gpt-compatible-model
```

## 8. 启动顺序

推荐顺序：

```bash
sudo systemctl enable --now python-ai-gateway.service
sudo systemctl start loong64-b1-upgrade.service
sudo systemctl enable --now loong64-b1-go.service
```

检查状态：

```bash
sudo systemctl status python-ai-gateway.service
sudo systemctl status loong64-b1-go.service
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

## 9. 前端托管

Go 主服务直接托管 PC Web 前端：

- `/api` 和 `/health` 仍由 Go 后端接口处理
- 其他浏览器路由走内嵌静态文件和 SPA fallback
- 默认监听 `HTTP_ADDR`，示例为 `0.0.0.0:8080`

如果学校统一入口需要 80/443、TLS 或统一域名，应在网关层单独评审和配置，不作为本项目默认交付依赖。

## 10. 验收留档

目标机部署后，至少记录：

- `uname -m`
- `/etc/os-release`
- `go version`
- `python3 --version`
- `pip freeze` 或等价依赖记录
- `systemctl status python-ai-gateway.service`
- `systemctl status loong64-b1-go.service`
- `/health/live`
- `/health/ready`

完整清单见：

- `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
