# loong64-b1-go

基于大模型技术的软件实训教学结果检查评价与报表系统。该仓库面向 LoongArch 架构 + 银河麒麟高级服务器版交付，当前采用 `Go 主服务 + Python 推理微服务` 的双服务架构，不复用旧 TypeScript 仓库。

## 目标

- 支持本地或云端 OpenAI-compatible 大模型服务。
- 提供 PC Web 可视化界面，覆盖学生、教师、管理员工作台。
- 支持 Word、PDF、报告、截图、代码包、Git 链接等实训成果上传与解析。
- 实现实训要求核查、步骤完整性检查、逻辑风险初筛和多维指标评价。
- 保留教师主观评分、改分理由、评语和最终发布入口。
- 生成个人评价报告、班级/课程统计报表，支持 Excel/PDF 导出。
- 固定适配 LoongArch + 银河麒麟高级服务器版部署。

## 当前状态

阶段 7 已完成；阶段 7.5 已推进到首个完整可用版本；阶段 7.6 已进入首个可用版本；阶段 7.7 已进入首个可用版本；阶段 8 的当前安全收口切片已进入可用版本。当前主线已达到试点 MVP 交付标准：支持前端初始化搭建、上传/核查/初评/复核/发布闭环，以及 HTML/CSV/XLSX/PDF 报表导出。

同时，AI 相关实现已经明确拆分为：

- `Go`：业务主系统、权限、教学流程、审计、报表、对象存储、数据库真相源
- `Python`：推理微服务、附件解析增强、初评、检索、结构化输出清洗、本地模型适配

已包含：

- `AGENT.md`：固定开发流程和 GitHub MCP 远端提交流程。
- `PLAN.md`：完整开发计划、阶段验收和 LoongArch 检查清单。
- `cmd/server`：Go HTTP 服务、请求日志、请求 ID、panic recovery、live/ready 健康检查。
- `cmd/migrate`：支持 PostgreSQL / SQLite 的迁移命令。
- `internal/storage`：本地 ObjectStore 初始化和健康检查。
- `internal/jobs`：基础 Job 状态模型和内存执行器。
- `internal/teaching`：用户、课程、班级、选课、评价模板版本、实训任务、提交、附件、规则核查、初评、教师复核发布和报表导出服务。
- `internal/aigateway`：Go 到 Python 推理微服务的内部 HTTP client。
- `.github/workflows`：Auto Build、自动代码审核与 CD 发布流水线。
- `.github/workflows/container-smoke.yml`：容器次级交付的构建与启动冒烟验证。
- `deploy/kylin`：银河麒麟 systemd 部署骨架和冒烟测试脚本。
- `deploy/kylin/nginx`：银河麒麟静态站点与反向代理示例。
- `Containerfile` / `compose.yaml`：容器次级交付资产，默认用于开发、演示和 CI 冒烟。
- `docs/SINGLE_BINARY_RUNTIME.md`：单二进制托管前端与默认 SQLite 方案。
- `docs/DEPLOYMENT_ASSISTANT.md`：部署助手、上下文快照与工具确认说明。
- `docs/PYTHON_AI_MIDDLEWARE.md`：Python 推理微服务职责、调试方式与部署边界。
- `internal/authn`：登录、会话、cookie 与 actor 解析。
- `internal/teaching/assets/fonts/NotoSansSC-VF.ttf`：内嵌 PDF 中文字体资产。
- 管理员可通过后端 API 和 PC Web 用户管理卡片为现有用户设置密码。
- 已登录用户可通过后端 API 和 PC Web 账号安全卡片修改当前密码；修改后需要重新登录。
- 管理员与教师可通过前端最小搭建面板创建用户、班级、课程、模板版本和实验任务。
- `scripts/dev`：本地 PostgreSQL 初始化和启动脚本。
- `api/openapi.yaml`：API 说明，当前版本 0.8.0。
- `docs/`：安全基线、LoongArch 兼容性记录、CD 流水线、部署和本地 PostgreSQL 说明。
- `web/`：Vue 3 + Vite + TypeScript PC Web MVP。

## 当前架构

当前推荐架构是 4 个核心运行单元：

1. `Web 前端`
2. `Go 主服务`
3. `Python 推理微服务`
4. `数据库 / 对象存储 / 模型服务`

职责边界：

- `Go 主服务`
  - 对外 API
  - 登录 / 会话 / 权限
  - 教学业务主流程
  - 审计日志
  - 报表与导出
  - 数据库和对象存储写入
- `Python 推理微服务`
  - `parse-artifact`
  - `evaluate-submission`
  - `build-retrieval-index`
  - `query-retrieval`
  - 本地模型 / OpenAI-compatible 模型调用
  - 检索上下文和结构化输出整理

也就是说：

- `Go 管业务`
- `Python 管推理`

## 推荐运行方式

开发态推荐同时启动：

- Go 主服务
- Python 推理微服务
- SQLite 或 PostgreSQL
- 本地模型服务或远端模型网关

试点 / 生产态推荐部署为：

- `loong64-b1-go.service`
- `python-ai-gateway.service`

如果不启用 Python 微服务，Go 侧仍能保留部分回退能力，但当前推荐路径已经是双服务协作。

## 快速启动

```bash
go run ./cmd/server
```

默认监听：`http://127.0.0.1:8080`

当前默认数据库模式是本地 SQLite：

```bash
RUNTIME_CONFIG_PATH=./config/runtime.json
DB_DRIVER=sqlite
SQLITE_PATH=./data/loong64-b1-go.db
AUTO_MIGRATE=true
```

当前主链路认证方式：

```bash
POST /api/v1/auth/login
POST /api/v1/auth/logout
PUT  /api/v1/auth/password
GET  /api/v1/me
PUT  /api/v1/admin/users/{userID}/password
```

浏览器端使用 `httpOnly` session cookie；`X-Actor-ID / X-Actor-Roles` 只保留给 `DEV_AUTH_BYPASS=true` 的开发态本机调试。

如果要启用完整 AI 链路，建议同时启动 Python 推理微服务：

```bash
cd python-ai-gateway
python -m venv .venv
. .venv/bin/activate
pip install -e .
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Windows PowerShell：

```powershell
cd python-ai-gateway
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -e .
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

然后在 Go 侧配置：

```bash
AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_TIMEOUT=10s
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

如果需要构建可直接托管前端页面的完整二进制，先构建前端，再使用 `webui` build tag：

```bash
npm ci --prefix web
npm run build --prefix web
go build -tags webui ./cmd/server
```

健康检查：

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

`/health/ready` 会检查当前数据库驱动和本地对象存储。默认 SQLite 模式下，只要数据库文件可打开且存储目录可写，`ready` 应返回 `200`。

管理员可通过 PC Web 中的“运行配置”卡片保存 `sqlite / postgres` 数据库模式和连接参数；后端会写入 `runtime.json`，并明确提示需要重启服务生效。
如果数据库中还没有任何用户，PC Web 会优先进入 bootstrap 卡片，允许直接创建首个管理员。

## 数据库迁移

默认 SQLite：

```bash
DB_DRIVER=sqlite SQLITE_PATH=./data/loong64-b1-go.db go run ./cmd/migrate up
```

默认情况下，`cmd/server` 在 SQLite 模式下会自动执行迁移，因此本地直接运行服务二进制即可初始化数据库；`cmd/migrate` 仍保留给显式迁移、脚本化部署和 PostgreSQL 场景。

PostgreSQL：

```bash
DB_DRIVER=postgres DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable go run ./cmd/migrate up
```

Windows PowerShell：

```powershell
$env:DB_DRIVER='sqlite'; $env:SQLITE_PATH='./data/loong64-b1-go.db'; go run ./cmd/migrate up
$env:DB_DRIVER='postgres'; $env:DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable'; go run ./cmd/migrate up
```

## 可选环境变量

```bash
HTTP_ADDR=127.0.0.1:8080
APP_ENV=development
DEV_AUTH_BYPASS=false
SESSION_COOKIE_NAME=loong64_b1_session
CSRF_COOKIE_NAME=loong64_b1_csrf
SESSION_TTL=168h
SESSION_REFRESH_INTERVAL=15m
SESSION_CLEANUP_INTERVAL=1h
SESSION_SECURE_COOKIE=false
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_SHUTDOWN_TIMEOUT=10s
READY_TIMEOUT=2s
STORAGE_ROOT=./storage
RUNTIME_CONFIG_PATH=./config/runtime.json
MIGRATIONS_DIR=migrations
DB_DRIVER=sqlite
SQLITE_PATH=./data/loong64-b1-go.db
AUTO_MIGRATE=true
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable
DB_MAX_CONNS=10
MAX_UPLOAD_BYTES=52428800
MAX_ARTIFACTS_PER_SUBMISSION=20
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
LLM_TIMEOUT=30s
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

- Auto Build：每次 push、PR 或手动触发时运行格式检查、Go 测试、Web 构建和 loong64 目标构建，并组装 3 个用户向 bundle：`full`、`backend`、`frontend`。
- Code Quality Review：每次 push、PR 或手动触发时运行 `golangci-lint` 与前端构建；PR 中配置 `SOURCERY_TOKEN` 后自动运行 SourceryAI 代码审核。
- CD Publish Artifacts：`main` 分支 Auto Build 成功后自动下载其产物并创建 GitHub Release，也支持手动输入 run id 发布指定构建。

详见 `docs/CD_PIPELINE.md` 和 `docs/CODE_REVIEW_CI.md`。

## 教学域与上传 API

`/api/v1` 接口覆盖管理员维护用户/班级/课程/选课、教师维护评价模板和实训任务、学生创建提交并上传成果附件。

阶段 3/4 支持：

- 学生创建提交：`POST /api/v1/student/experiments/{experimentID}/submissions`
- 学生上传附件：`POST /api/v1/student/submissions/{submissionID}/artifacts`
- 学生登记 Git 链接：`POST /api/v1/student/submissions/{submissionID}/artifact-links`
- 教师查看提交：`GET /api/v1/teacher/experiments/{experimentID}/submissions`
- 教师触发规则核查/LLM 初评：`POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial`
- 教师查看最新初评：`GET /api/v1/teacher/submissions/{submissionID}/evaluations/latest`
- 教师保存复核草稿：`PUT /api/v1/teacher/submissions/{submissionID}/review`
- 教师发布最终评价：`POST /api/v1/teacher/submissions/{submissionID}/review/publish`
- 学生查看已发布评价：`GET /api/v1/student/submissions/{submissionID}/review`
- 教师查看个人评价报告：`GET /api/v1/teacher/submissions/{submissionID}/report`
- 学生查看已发布个人报告：`GET /api/v1/student/submissions/{submissionID}/report`
- 教师查看实验统计：`GET /api/v1/teacher/experiments/{experimentID}/reports/summary`
- 教师查看课程统计：`GET /api/v1/teacher/courses/{courseID}/reports/summary`
- 教师导出个人报告：`POST /api/v1/teacher/submissions/{submissionID}/report-exports`
- 教师导出实验统计：`POST /api/v1/teacher/experiments/{experimentID}/report-exports`
- 教师导出课程统计：`POST /api/v1/teacher/courses/{courseID}/report-exports`
- 教师查询/下载导出：`GET /api/v1/teacher/report-exports/{exportID}` 与 `/download`

上传文件会计算 SHA-256、保存对象存储 key、登记附件元数据和创建 `submission_artifact_parse` 解析任务；当前实现已支持文本/Markdown 摘要、DOCX 正文摘要、PDF 基础文本摘要、截图尺寸和 ZIP 清单摘要。阶段 4 初评只生成教师复核用建议结果，不写最终成绩、不向学生发布；阶段 5 发布后的教师最终评价对学生可见且不可被后台初评覆盖。阶段 6 的 HTML 报表是规范归档源，CSV 带 UTF-8 BOM 便于 WPS/Excel/LibreOffice 打开；现已补齐课程跨实验统计，以及 CSV/XLSX/PDF 导出。PDF 使用纯 Go 生成与内嵌中文字体，不依赖目标机浏览器或 office 套件。

开发态仍可在 `DEV_AUTH_BYPASS=true` 时使用请求头标识操作者：

```bash
X-Actor-ID: teacher-1
X-Actor-Roles: teacher
```

详见 `docs/TEACHING_DOMAIN.md`、`docs/SUBMISSION_UPLOAD.md`、`docs/VERIFICATION_EVALUATION.md`、`docs/TEACHER_REVIEW.md` 和 `api/openapi.yaml`。


## 前端开发

```bash
cd web
npm ci
npm run dev
npm run lint
npm run build
```

开发服务器默认代理 `/api` 和 `/health` 到 `http://127.0.0.1:8080`。主链路默认使用 session 登录；只有在本机且启用 `DEV_AUTH_BYPASS=true` 时，才建议继续通过 `X-Actor-ID` / `X-Actor-Roles` 做调试。

## Python 推理微服务调试

当前 Python 微服务已支持：

- 附件解析
- 初评
- 检索索引
- 检索查询

本地调试时优先看：

- `python-ai-gateway/ai_gateway/app.py`
- `python-ai-gateway/ai_gateway/parser.py`
- `python-ai-gateway/ai_gateway/evaluator.py`
- `python-ai-gateway/ai_gateway/retrieval.py`

推荐的最小调试命令：

```bash
python -m py_compile python-ai-gateway/ai_gateway/app.py python-ai-gateway/ai_gateway/models.py python-ai-gateway/ai_gateway/parser.py python-ai-gateway/ai_gateway/evaluator.py python-ai-gateway/ai_gateway/retrieval.py
```

## 部署与本地数据库

- 银河麒麟 systemd 部署：`docs/DEPLOY_KYLIN.md`
- Python 推理微服务说明：`docs/PYTHON_AI_MIDDLEWARE.md`
- Stage 7 部署验证清单：`docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
- 默认 SQLite / PostgreSQL 运行方案：`docs/SINGLE_BINARY_RUNTIME.md`
- MVP 交付清单：`docs/MVP_DELIVERY.md`
- UAT 执行手册与记录模板：`docs/UAT_CHECKLIST.md`
- 部署助手与上下文工程：`docs/DEPLOYMENT_ASSISTANT.md`
- 容器次级交付：`docs/CONTAINER_RUNTIME.md`
- 本地 PostgreSQL 调试：`docs/LOCAL_POSTGRES.md`

## 目录规划

```text
.github/workflows/     Auto Build、代码审核与 CD 发布流水线
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
