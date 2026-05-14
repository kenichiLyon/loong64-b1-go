# 银河麒麟 + LoongArch 部署说明

当前推荐部署形态已经不是“只跑一个 Go 服务”。

基于现在的项目结构，推荐的部署方式是：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

也就是：

- `Go 主服务`
- `Python 推理微服务`

共同组成正式交付。
# 银河麒麟 + LoongArch 部署说明

当前推荐部署形态已经不是“只跑一个 Go 服务”。

基于现在的项目结构，推荐的部署方式是：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

也就是：

- `Go 主服务`
- `Python 推理微服务`

共同组成正式交付。

## 1. 发布产物

CD 发布后，优先下载：
CD 发布后，优先下载：

- `loong64-b1-go-full-linux-loong64.tar.gz`

如果需要拆分部署，也可以使用：


如果需要拆分部署，也可以使用：

- `loong64-b1-go-backend-linux-loong64.tar.gz`
- `loong64-b1-go-frontend.tar.gz`

建议先校验：
建议先校验：

```bash
sha256sum loong64-b1-go-full-linux-loong64.tar.gz
```

## 2. 目标目录
## 2. 目标目录

推荐目录：
推荐目录：

```text
/opt/loong64-b1-go/bin
/opt/loong64-b1-go/python-ai-gateway
/opt/loong64-b1-go/python-ai-gateway
/etc/loong64-b1-go/loong64-b1-go.env
/etc/loong64-b1-go/python-ai-gateway.env
/etc/loong64-b1-go/python-ai-gateway.env
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

- 用户：`loong64b1`
- `loong64-b1-go.service`
- `loong64-b1-migrate.service`
- `python-ai-gateway.service`
- `/etc/loong64-b1-go/loong64-b1-go.env`
- `/etc/loong64-b1-go/python-ai-gateway.env`
- 用户：`loong64b1`
- `loong64-b1-go.service`
- `loong64-b1-migrate.service`
- `python-ai-gateway.service`
- `/etc/loong64-b1-go/loong64-b1-go.env`
- `/etc/loong64-b1-go/python-ai-gateway.env`

## 4. 部署 Go 主服务

解压 full bundle：
## 4. 部署 Go 主服务

解压 full bundle：

```bash
sudo install -d -m 0755 /opt/loong64-b1-go
sudo tar -xzf loong64-b1-go-full-linux-loong64.tar.gz -C /opt/loong64-b1-go
```

如果使用 full bundle，建议安装完整二进制为主程序：
如果使用 full bundle，建议安装完整二进制为主程序：

```bash
sudo install -o loong64b1 -g loong64b1 -m 0755 /opt/loong64-b1-go/bin/loong64-b1-go-linux-loong64-full /opt/loong64-b1-go/bin/loong64-b1-go-linux-loong64
```

## 5. 部署 Python 推理微服务

Python 微服务代码目录：
## 5. 部署 Python 推理微服务

Python 微服务代码目录：

```text
/opt/loong64-b1-go/python-ai-gateway
```

推荐做法：

1. 把仓库中的 `python-ai-gateway/` 目录同步到目标机
2. 在目标机创建 venv
3. 安装 Python 依赖

示例：
```text
/opt/loong64-b1-go/python-ai-gateway
```

推荐做法：

1. 把仓库中的 `python-ai-gateway/` 目录同步到目标机
2. 在目标机创建 venv
3. 安装 Python 依赖

示例：

```bash
cd /opt/loong64-b1-go/python-ai-gateway
python3 -m venv .venv
. .venv/bin/activate
pip install -e .
```

如果目标机没有外网：

- 需要预先准备 Python wheel 缓存或内网源
- 这是部署时必须留档的事项

## 6. Go 环境变量

编辑：

```text
/etc/loong64-b1-go/loong64-b1-go.env
```

至少确认：

```env
DB_DRIVER=sqlite
SQLITE_PATH=/var/lib/loong64-b1-go/data/loong64-b1-go.db

AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_API_KEY=
AI_GATEWAY_TIMEOUT=10s

AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_API_KEY=
AI_GATEWAY_TIMEOUT=10s
```

如果使用 PostgreSQL：
如果使用 PostgreSQL：

```env
DB_DRIVER=postgres
DATABASE_URL=postgres://loong64_b1:CHANGE_ME@127.0.0.1:5432/loong64_b1?sslmode=disable
```

## 7. Python 环境变量

编辑：

```text
/etc/loong64-b1-go/python-ai-gateway.env
```

至少确认：
## 7. Python 环境变量

编辑：

```text
/etc/loong64-b1-go/python-ai-gateway.env
```

至少确认：

```env
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

如果 Python 微服务要接本地模型服务：
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

如果 Python 微服务要接本地模型服务：

```env
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
```

如果要接校内或云端模型网关：
如果要接校内或云端模型网关：

```env
AI_GATEWAY_LLM_BASE_URL=https://llm-gateway.example.edu/v1
AI_GATEWAY_LLM_API_KEY=REDACTED
AI_GATEWAY_LLM_MODEL=gpt-compatible-model
AI_GATEWAY_LLM_BASE_URL=https://llm-gateway.example.edu/v1
AI_GATEWAY_LLM_API_KEY=REDACTED
AI_GATEWAY_LLM_MODEL=gpt-compatible-model
```

## 8. 启动顺序

推荐顺序：
## 8. 启动顺序

推荐顺序：

```bash
sudo systemctl enable --now python-ai-gateway.service
sudo systemctl enable --now python-ai-gateway.service
sudo systemctl start loong64-b1-migrate.service
sudo systemctl enable --now loong64-b1-go.service
```

检查状态：
检查状态：

```bash
sudo systemctl status python-ai-gateway.service
sudo systemctl status loong64-b1-go.service
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

## 9. 前端托管

如果使用 full bundle：

- Go 主服务直接托管前端

如果使用分离部署：

- 使用 Nginx / 学校统一网关托管静态资源
- `/api` 和 `/health` 反代到 Go 主服务

Nginx 示例见：

- `deploy/kylin/nginx/loong64-b1-go.conf.example`

## 10. 当前部署结论

现在的部署方式应当理解成：

- `Go 是业务服务`
- `Python 是推理服务`


## 11. 验收与留档

完整验收仍要记录：
sudo systemctl status python-ai-gateway.service
sudo systemctl status loong64-b1-go.service
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

## 9. 前端托管

如果使用 full bundle：

- Go 主服务直接托管前端

如果使用分离部署：

- 使用 Nginx / 学校统一网关托管静态资源
- `/api` 和 `/health` 反代到 Go 主服务

Nginx 示例见：

- `deploy/kylin/nginx/loong64-b1-go.conf.example`

## 10. 当前部署结论

现在的部署方式应当理解成：

- `Go 是业务服务`
- `Python 是推理服务`


## 11. 验收与留档

完整验收仍要记录：

- `uname -m`
- `uname -m`
- `/etc/os-release`
- `go version`
- `python --version`
- `pip freeze` 或等价依赖记录
- `systemctl status python-ai-gateway.service`
- `python --version`
- `pip freeze` 或等价依赖记录
- `systemctl status python-ai-gateway.service`
- `systemctl status loong64-b1-go.service`
- `/health/live` 与 `/health/ready`

完整清单见：

- `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
- `/health/live` 与 `/health/ready`

完整清单见：

- `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
