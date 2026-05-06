# Stage 7 部署验证

本文件用于记录 LoongArch + 银河麒麟高级服务器版上的部署验收结果。目标不是单次“能跑起来”，而是形成可重复、可留档的验收流程。

## 1. 验证范围

- release 二进制与 Web 静态产物完整性
- systemd 单元安装与服务拉起
- PostgreSQL 迁移、备份与恢复命令
- `/health/live`、`/health/ready` 健康检查
- 目标机架构、字体、systemd、数据库与服务状态采样

## 2. 推荐执行顺序

```bash
sha256sum -c SHA256SUMS
sudo sh deploy/kylin/scripts/install-systemd.sh
sudo systemctl start loong64-b1-migrate.service
sudo systemctl enable --now loong64-b1-go.service
sh deploy/kylin/scripts/preflight-check.sh
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/verify-deployment.sh
sh deploy/kylin/scripts/collect-env.sh /tmp/loong64-b1-go-stage7.txt
```

## 3. 备份与恢复

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

## 4. LLM 配置示例

本地模型服务：

```env
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
LLM_API_KEY=
```

云端或校内网关：

```env
LLM_BASE_URL=https://llm-gateway.example.edu/v1
LLM_MODEL=gpt-compatible-model
LLM_API_KEY=REDACTED
```

## 5. 验收记录模板

```text
日期：
机器：
架构：
银河麒麟版本：
内核：
Go：
PostgreSQL：
字体检测：
服务状态：
验证命令：
结果：
问题与处理：
```
