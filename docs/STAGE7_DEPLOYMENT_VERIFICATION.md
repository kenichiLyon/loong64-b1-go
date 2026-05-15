# Stage 7 部署验证

本文件用于记录 LoongArch + 银河麒麟高级服务器版上的部署验收结果。目标不是单次“能跑起来”，而是形成可重复、可留档的验收流程。

## 1. 验证范围

- release 二进制与 Web 静态产物完整性
- Go 二进制内嵌 PC Web 前端
- systemd 单元安装与服务拉起
- 系统升级迁移与服务启动
- `/health/live`、`/health/ready` 健康检查
- 目标机架构、字体、systemd、数据库与服务状态采样

## 2. 推荐执行顺序

按下面顺序执行，并把输出保存在验收记录中：

```bash
sha256sum -c SHA256SUMS
sudo sh deploy/kylin/scripts/install-systemd.sh
sh deploy/kylin/scripts/preflight-check.sh
sudo systemctl start loong64-b1-upgrade.service
sudo systemctl enable --now loong64-b1-go.service
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/smoke-test.sh
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/verify-deployment.sh
BASE_URL=http://127.0.0.1:8080 sh deploy/kylin/scripts/collect-env.sh /tmp/loong64-b1-go-stage7.txt
```

## 3. 每步期望结果

1. `sha256sum -c SHA256SUMS`
   - 所有发布物返回 `OK`
2. `install-systemd.sh`
   - 创建用户、组、目录、unit 文件和 env 模板
3. `preflight-check.sh`
   - 返回 `Preflight check passed.`
4. `loong64-b1-upgrade.service`
   - 启动成功，无升级迁移报错
5. `loong64-b1-go.service`
   - `active (running)`
6. `smoke-test.sh`
   - `/health/live` 和 `/health/ready` 都返回成功
7. `verify-deployment.sh`
   - 根路径返回内嵌 PC Web HTML，健康检查返回正确 JSON
8. `collect-env.sh`
   - 生成包含系统、版本、systemd、health 输出的记录文件

## 4. 备份与恢复

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

## 5. LLM 配置示例

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

## 6. 验收记录模板

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
执行步骤：
关键命令输出：
结果：
问题与处理：
```
