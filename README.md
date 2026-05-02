# loong64-b1-go

基于大模型技术的软件实训教学结果检查评价与报表系统。该仓库是面向 LoongArch 架构 + 银河麒麟高级服务器版的 Go 技术路线实现，不复用旧 TypeScript 仓库。

## 目标

- 支持本地或云端 OpenAI-compatible 大模型服务。
- 提供 PC Web 可视化界面，覆盖学生、教师、管理员工作台。
- 支持 Word、PDF、报告、截图、代码包、Git 链接等实训成果上传与解析。
- 实现实训要求核查、步骤完整性检查、逻辑风险初筛和多维指标评价。
- 保留教师主观评分、改分理由、评语和最终发布入口。
- 生成个人评价报告、班级/课程统计报表，支持 Excel/PDF 导出。
- 固定适配 LoongArch + 银河麒麟高级服务器版部署。

## 当前状态

阶段 1.5：后端基础设施 + 部署骨架。

已包含：

- `AGENT.md`：固定开发流程和 GitHub MCP 远端提交流程。
- `PLAN.md`：完整开发计划、阶段验收和 LoongArch 检查清单。
- `cmd/server`：Go HTTP 服务、请求日志、请求 ID、panic recovery、live/ready 健康检查。
- `cmd/migrate`：PostgreSQL 迁移命令。
- `internal/storage`：本地 ObjectStore 初始化和健康检查。
- `internal/jobs`：基础 Job 状态模型和内存执行器。
- `.github/workflows`：Auto Build 与 CD 发布流水线。
- `deploy/kylin`：银河麒麟 systemd 部署骨架和冒烟测试脚本。
- `scripts/dev`：本地 PostgreSQL 初始化和启动脚本。
- `api/openapi.yaml`：初始 API 说明。
- `docs/`：安全基线、LoongArch 兼容性记录、CD 流水线、部署和本地 PostgreSQL 说明。

## 快速启动

```bash
go run ./cmd/server
```

默认监听：`http://127.0.0.1:8080`

健康检查：

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

`/health/ready` 会检查 PostgreSQL 和本地对象存储；未设置 `DATABASE_URL` 时会返回 503，这是预期行为。

## 数据库迁移

```bash
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable go run ./cmd/migrate up
```

Windows PowerShell：

```powershell
$env:DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable'; go run ./cmd/migrate up
```

## 可选环境变量

```bash
HTTP_ADDR=127.0.0.1:8080
APP_ENV=development
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_SHUTDOWN_TIMEOUT=10s
READY_TIMEOUT=2s
STORAGE_ROOT=./storage
MIGRATIONS_DIR=migrations
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable
DB_MAX_CONNS=10
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
```

## 验证

```bash
gofmt -w cmd internal
go test ./...
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate
```

Windows PowerShell 交叉编译示例：

```powershell
$env:GOOS='linux'; $env:GOARCH='loong64'; $env:CGO_ENABLED='0'; go build ./cmd/server; go build ./cmd/migrate; Remove-Item Env:GOOS,Env:GOARCH,Env:CGO_ENABLED
```

## CI/CD

- Auto Build：每次 push、PR 或手动触发时运行格式检查、测试、linux/amd64 和 linux/loong64 构建，并上传构建产物。
- CD Publish Artifacts：`main` 分支 Auto Build 成功后自动下载其产物并创建 GitHub Release，也支持手动输入 run id 发布指定构建。

详见 `docs/CD_PIPELINE.md`。

## 部署与本地数据库

- 银河麒麟 systemd 部署：`docs/DEPLOY_KYLIN.md`
- 本地 PostgreSQL 调试：`docs/LOCAL_POSTGRES.md`

## 目录规划

```text
.github/workflows/     Auto Build 与 CD 发布流水线
cmd/                   Go 程序入口
internal/              后端内部模块
api/                   OpenAPI 和接口契约
migrations/            PostgreSQL 迁移脚本
web/                   PC Web 前端
deploy/kylin/          银河麒麟部署脚本和 systemd 文件
docs/                  架构、安全、兼容性和用户文档
testdata/              脱敏或合成测试样例
```

## 版本控制纪律

本项目遵循 `AGENT.md`：任何仓库内容修改都必须通过 GitHub MCP 提交到远端 `kenichiLyon/loong64-b1-go`，并在最终回复中说明 commit SHA 和验证结果。
