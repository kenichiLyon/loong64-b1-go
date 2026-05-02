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

阶段 0：仓库治理与最小 Go 服务骨架。

已包含：

- `AGENT.md`：固定开发流程和 GitHub MCP 远端提交流程。
- `PLAN.md`：完整开发计划、阶段验收和 LoongArch 检查清单。
- `cmd/server`：最小健康检查服务。
- `api/openapi.yaml`：初始 API 说明。
- `docs/`：安全基线和 LoongArch 兼容性记录。

## 快速启动

```bash
go run ./cmd/server
```

默认监听：`http://127.0.0.1:8080`

健康检查：

```bash
curl http://127.0.0.1:8080/health
```

可选环境变量：

```bash
HTTP_ADDR=127.0.0.1:8080
APP_ENV=development
STORAGE_ROOT=./storage
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
```

## 验证

```bash
gofmt -w cmd internal
go test ./...
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
```

Windows PowerShell 交叉编译示例：

```powershell
$env:GOOS='linux'; $env:GOARCH='loong64'; $env:CGO_ENABLED='0'; go build ./cmd/server; Remove-Item Env:GOOS,Env:GOARCH,Env:CGO_ENABLED
```

## 目录规划

```text
cmd/                  Go 程序入口
internal/             后端内部模块
api/                  OpenAPI 和接口契约
migrations/           PostgreSQL 迁移脚本
web/                  PC Web 前端
frontend/             如后续改名，需同步 PLAN.md
deploy/kylin/         银河麒麟部署脚本和 systemd 文件
docs/                 架构、安全、兼容性和用户文档
testdata/             脱敏或合成测试样例
```

## 版本控制纪律

本项目遵循 `AGENT.md`：任何仓库内容修改都必须通过 GitHub MCP 提交到远端 `kenichiLyon/loong64-b1-go`，并在最终回复中说明 commit SHA 和验证结果。
