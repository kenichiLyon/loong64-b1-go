# 项目架构梳理

本文从代码结构、运行拓扑、核心模块、数据模型和主业务链路几个层面说明当前项目架构。更细的专项说明仍以各专题文档为准，例如部署见 `docs/DEPLOY_KYLIN.md`，Python AI Worker 见 `docs/PYTHON_AI_MIDDLEWARE.md`，安全见 `docs/SECURITY.md`。

## 1. 总体定位

本项目是“基于大模型技术的软件实训教学结果检查评价与报表系统”，当前面向 `LoongArch + 银河麒麟高级服务器版` 交付。

架构原则：

- `Go 主服务`负责业务闭环、权限、数据一致性、审计、报表与对外 API。
- `Python AI Worker`负责附件解析、检索上下文、RAG 组装和模型推理相关能力；当前实现目录和服务名仍使用 `python-ai-gateway`。
- `Web 前端`是 PC 工作台，默认内嵌在 Go 主服务二进制中，覆盖管理员、教师、学生与部署初始化流程。
- `数据库 + 本地对象存储`保存业务状态、审计记录、附件和报表导出文件。

一句话边界：

```text
Go 管业务和 Web 入口，Python AI Worker 管推理，DB/Storage 管状态与文件。
```

## 2. 运行拓扑

推荐生产/试点拓扑是双服务：

```text
浏览器
  |
  |  HTTP / Cookie / CSRF
  v
Go 主服务 loong64-b1-go
  |-- /api/v1/*             对外业务 API
  |-- /health/*             存活和就绪检查
  |-- embedded web/dist     PC Web UI 与 SPA fallback
  |
  |-- PostgreSQL 或 SQLite  业务数据、审计、会话、助手上下文
  |-- LocalStore            artifacts/ reports/ tmp/
  |
  |-- Python AI Worker      附件解析、提交初评、检索
  |
  `-- OpenAI-compatible LLM 可选直连，用于 Go 侧初评/部署助手

Python AI Worker 当前实现 python-ai-gateway
  |
  |-- /internal/parse-artifact
  |-- /internal/evaluate-submission
  |-- /internal/build-retrieval-index
  `-- /internal/query-retrieval
       |
       `-- OpenAI-compatible LLM，可为本地模型、校内网关或云端服务
```

说明：

- Go 主服务可不启用 Python AI Worker，但当前正式推荐路径是双服务协作。
- 当 Go 配置了 `AI_GATEWAY_BASE_URL` 时，Go 会把 Python AI Worker 纳入 readiness 检查，并优先使用它进行附件解析和 AI 初评。
- Go 侧也保留 `LLM_BASE_URL` 直连 OpenAI-compatible 模型的能力，可用于初评兜底和部署助手。
- Python AI Worker 自身使用 `AI_GATEWAY_LLM_*` 环境变量连接模型服务。

## 3. 仓库目录

```text
.github/workflows/       CI、代码审核和发布流水线
api/openapi.yaml         OpenAPI 接口描述
cmd/server/              Go 主服务入口
cmd/upgrade/             系统升级迁移命令入口
internal/api/            HTTP 路由、认证入口、bootstrap、运行配置和部署助手 API
internal/authn/          登录、session、CSRF、密码修改和会话仓储
internal/teaching/       教学主领域：用户、课程、提交、解析、初评、复核、报表
internal/assistant/      部署助手会话、上下文快照、受控工具调用和 LLM 调用日志
internal/database/       PostgreSQL / SQLite 统一数据库运行时抽象
internal/upgrade/        方言感知升级迁移执行器
internal/storage/        本地对象存储封装
internal/aigateway/      Go 调 Python AI Worker 当前实现的客户端
internal/llm/            OpenAI-compatible Chat Completions 客户端
internal/health/         live/ready 健康检查组合器
internal/httpx/          JSON 响应、错误响应、中间件和访问日志
internal/jobs/           作业模型和内存 Runner，给后续异步 worker 扩展预留
migrations/              PostgreSQL 升级迁移 SQL
migrations/sqlite/       SQLite 升级迁移 SQL
python-ai-gateway/       Python AI Worker 当前实现
web/                     Vue + Vite PC Web 前端
deploy/kylin/            银河麒麟 systemd、env 和验收脚本
scripts/                 本地开发、UAT、发布辅助脚本
docs/                    架构、部署、安全、UAT、专项功能文档
```

## 4. Go 主服务分层

Go 侧采用较清晰的“HTTP 适配层 -> 领域服务 -> 仓储接口 -> 数据库实现”分层。

```text
cmd/server/main.go
  |
  v
internal/api.NewHandler
  |
  |-- authHandler / bootstrapHandler / runtimeConfigHandler
  |-- teaching.RegisterRoutes
  |-- deploymentAssistantHandler
  |
  v
internal/teaching.Service
  |
  |-- Repository interface
  |-- ArtifactStore interface
  |-- ArtifactParser interface
  |-- SubmissionEvaluator interface
  `-- LLMCompleter interface
       |
       |-- PostgresRepository / SQLiteRepository
       |-- storage.LocalStore
       |-- aigateway.Client
       `-- llm.Gateway
```

关键入口：

- `cmd/server/main.go`：加载配置、初始化本地存储、打开数据库、可选自动迁移、注册 HTTP handler、优雅退出。
- `internal/api/server.go`：组装所有依赖，注册 `/health`、认证、bootstrap、运行配置、部署助手和教学路由。
- `cmd/upgrade/main.go`：独立升级迁移命令，按 `DB_DRIVER` 选择 PostgreSQL 或 SQLite 迁移目录。
- `appembed_webui.go` / `appembed_stub.go`：通过 `webui` build tag 控制是否把 `web/dist` 内嵌到 Go 二进制。

### 4.1 配置层

配置由 `internal/config` 加载，优先环境变量，同时支持 `RUNTIME_CONFIG_PATH` 指向的本地 JSON 配置。

常用配置域：

- HTTP：`HTTP_ADDR`、`HTTP_READ_HEADER_TIMEOUT`、`HTTP_SHUTDOWN_TIMEOUT`。
- 数据库：`DB_DRIVER=sqlite|postgres`、`SQLITE_PATH`、`DATABASE_URL`、`AUTO_UPGRADE`。
- 存储：`STORAGE_ROOT`，默认 `./storage`。
- 会话：`SESSION_COOKIE_NAME`、`CSRF_COOKIE_NAME`、`SESSION_TTL`、`SESSION_SECURE_COOKIE`。
- Python AI Worker：`AI_GATEWAY_BASE_URL`、`AI_GATEWAY_API_KEY`、`AI_GATEWAY_TIMEOUT`。
- Go 直连 LLM：`LLM_BASE_URL`、`LLM_MODEL`、`LLM_API_KEY`、`LLM_TIMEOUT`。
- 上传限制：`MAX_UPLOAD_BYTES`、`MAX_ARTIFACTS_PER_SUBMISSION`。

运行配置文件由 `internal/runtimecfg` 管理，管理员可通过 Web 面板/API 修改。保存后的配置要求重启生效，不做运行期热切换。

### 4.2 数据库运行时与升级迁移

`internal/database.Pool` 统一包装：

- PostgreSQL：`pgxpool.Pool`。
- SQLite：`database/sql` + `modernc.org/sqlite`，并启用 `foreign_keys`、`busy_timeout`、`WAL`。

迁移属于系统发布和升级链路，不是 `internal/database` 的附属职责。数据库包只提供连接和运行时抽象；升级能力由服务启动时的 `AUTO_UPGRADE` 或独立 `cmd/upgrade` 调用 `internal/upgrade` 完成。

`internal/upgrade.Runner` 根据数据库驱动选择升级迁移目录：

- PostgreSQL：`migrations/*.sql`
- SQLite：`migrations/sqlite/*.sql`

迁移通过 `system_upgrades` 记录版本和 checksum，已应用迁移内容变化会报错。

### 4.3 HTTP 与中间件

主链路使用 `net/http.ServeMux`，路由按角色前缀划分：

- `/api/v1/admin/*`：管理员建用户、班级、课程、授权、运行配置。
- `/api/v1/teacher/*`：教师课程、模板、实验、提交查看、初评、复核、报表。
- `/api/v1/student/*`：学生实验列表、提交、附件、评价和报告查看。
- `/api/v1/bootstrap/*`：首次初始化和未初始化部署助手。
- `/api/v1/auth/*`：登录、登出、改密。

全局中间件链：

```text
Recover -> RequestID -> SessionRefresh -> AccessLog -> ServeMux
```

错误响应统一使用 `internal/httpx.WriteError`，领域错误通过 `teaching.ErrorKind` 映射到 HTTP 状态码。

## 5. 前端架构

前端位于 `web/`：

- 技术栈：Vue 3 + TypeScript + Vite。
- 入口：`web/src/main.ts`、`web/src/App.vue`。
- API 客户端：`web/src/lib/api.ts`。
- 类型定义：`web/src/lib/types.ts`。
- 主要面板组件：
  - `BootstrapPanel.vue`
  - `LoginPanel.vue`
  - `AdminSetupPanel.vue`
  - `AdminUserPanel.vue`
  - `TeacherSetupPanel.vue`
  - `SubmissionDetailPanel.vue`
  - `EvaluationPanel.vue`
  - `ReviewPanel.vue`
  - `ReportPanel.vue`
  - `RuntimeConfigPanel.vue`
  - `DeploymentAssistantPanel.vue`
  - `AccountSecurityPanel.vue`

开发态通过 Vite 代理：

```text
/api    -> http://127.0.0.1:8080
/health -> http://127.0.0.1:8080
```

发布态先执行 `npm run build --prefix web`，再用 `go build -tags webui ./cmd/server` 构建内嵌前端的完整二进制。正式交付只保留 Go 单入口。

## 6. Python AI Worker（当前 python-ai-gateway）

Python AI Worker 当前实现位于 `python-ai-gateway/`，使用 FastAPI。`AI Gateway` 是当前目录、配置和 systemd 单元沿用的兼容命名，后续升级应优先向更通用的 AI Worker 形态收敛。

核心文件：

- `ai_gateway/app.py`：FastAPI 路由和 bearer token 校验。
- `ai_gateway/models.py`：请求/响应 Pydantic 模型。
- `ai_gateway/parser.py`：本地文件解析，支持 txt/md、docx、pdf、图片元数据、zip manifest。
- `ai_gateway/retrieval.py`：有界内存检索索引，按 token overlap 查询。
- `ai_gateway/evaluator.py`：构造 RAG 上下文，调用 OpenAI-compatible 模型，校验结构化输出。

服务边界：

- 不持有业务主状态。
- 不发布成绩。
- 不直接暴露用户侧公开 API。
- 所有结果回到 Go，由 Go 负责持久化、权限、审计和人工复核。

当前检索是内存索引，适合试点和边界验证；后续如升级 embedding/向量库，应尽量保持现有内部接口不变。

## 7. 核心领域模型

数据库表按职责可分为几组。

### 7.1 基础设施表

- `app_metadata`：应用级元数据。
- `jobs`：作业记录，给解析、报表或后续 worker 扩展使用。
- `audit_logs`：操作审计。
- `system_upgrades`：迁移版本和 checksum。

注意：当前代码已有作业表和 `internal/jobs` Runner，但没有独立常驻 worker 进程读取数据库作业队列。上传解析和报表导出主要在请求链路内完成，并同步写入结果或导出文件。

### 7.2 用户与教学组织

- `users`、`user_roles`
- `classes`
- `courses`
- `course_classes`
- `course_teachers`
- `enrollments`

角色模型：

- `admin`：用户、班级、课程、授权、运行配置。
- `teacher`：被授权课程内的模板、实验、提交、复核、报表。
- `student`：自己的实验、提交、附件和已发布结果。

### 7.3 Rubric 与实验

- `rubric_templates`
- `rubric_template_versions`
- `rubric_metrics`
- `experiments`

设计要点：

- 实验绑定 `rubric_template_versions.id`，不绑定模板主表，保证历史评价可追溯。
- 指标权重用基点表示，`100% = 10000`。
- 支持 `strict_100` 和 `normalized` 两种权重模式。
- 已发布模板版本和指标禁止普通更新或删除。

### 7.4 提交与附件

- `submissions`
- `artifacts`
- `extracted_contents`

附件类型：

- `document`
- `report`
- `screenshot`
- `code_archive`
- `git_link`
- `other`

上传文件保存到 `STORAGE_ROOT/artifacts/...`，数据库只保存服务端生成的 `storage_key`、hash、元数据和解析摘要。Git 链接当前只登记 URL、commit 和备注，代码拉取/执行不在主 API 进程中完成。

### 7.5 初评、复核与发布

- `evaluation_results`
- `rule_check_findings`
- `metric_scores`
- `llm_call_logs`
- `teacher_reviews`
- `teacher_metric_scores`

设计要点：

- 规则核查和 LLM 初评只产出建议分、发现项、证据引用和置信度。
- 最终成绩由教师复核产生。
- 学生只能看到已发布评价。
- 已发布教师评价和逐项分数不可被草稿保存或后台初评覆盖。

### 7.6 报表导出

- `report_exports`

报表范围：

- `submission_report`
- `experiment_summary`
- `course_summary`

导出格式：

- `html`
- `csv`
- `xlsx`
- `pdf`

导出文件保存到 `STORAGE_ROOT/reports/...`，表内记录格式、状态、hash、大小、筛选条件和请求人。

### 7.7 部署助手

- `assistant_conversations`
- `assistant_messages`
- `assistant_context_snapshots`
- `assistant_tool_calls`
- `assistant_llm_calls`

部署助手按 scope 分为：

- `bootstrap`：系统未初始化时辅助选择配置、创建首个管理员。
- `deployment_admin`：管理员登录后辅助检查和修改运行配置。

受控工具包括检查 bootstrap 状态、查看运行配置、测试 SQLite 路径、测试 PostgreSQL 连接、保存运行配置、创建首个管理员。工具调用需要确认后执行，并会写入上下文快照和工具调用结果。

## 8. 主业务链路

### 8.1 首次启动与登录

```text
启动 Go 服务
  -> 加载 env/runtime.json
  -> 初始化 storage
  -> 打开 SQLite 或 PostgreSQL
  -> 可选自动迁移
  -> /api/v1/bootstrap/status
  -> 未初始化时创建首个管理员
  -> 登录创建 session cookie + CSRF cookie
```

认证设计：

- 主链路使用 httpOnly session cookie。
- 修改类请求使用 CSRF 双提交 cookie：cookie + `X-CSRF-Token`。
- 非生产环境可用 `DEV_AUTH_BYPASS=true` 做 loopback 本机冒烟，但生产禁止依赖开发头。

### 8.2 管理员搭建教学基础数据

```text
创建用户/角色
  -> 创建班级
  -> 创建课程
  -> 关联课程班级
  -> 分配教师
  -> 登记学生选课
```

该链路主要落在 `internal/teaching/service.go` 和 `internal/teaching/{postgres,sqlite}.go`。

### 8.3 教师搭建评价模板与实验

```text
创建 rubric template
  -> 创建 rubric version + metrics
  -> 发布 rubric version
  -> 创建 experiment 并绑定已发布 version
  -> 发布 experiment
```

发布后的模板版本用于后续提交评价，避免指标被修改导致历史评分不可追溯。

### 8.4 学生提交成果

```text
学生查看已发布实验
  -> 创建 submission
  -> 上传文件或登记 Git 链接
  -> 文件写入 LocalStore
  -> Go 可同步调用 Python parse-artifact
  -> artifacts/extracted_contents 持久化
```

上传安全边界：

- 限制文件大小和单次提交附件数量。
- 服务端生成存储 key。
- 写入前计算 SHA-256。
- 不信任客户端文件名作为路径。
- 学生代码不在 API 进程内执行。

### 8.5 规则核查与 AI 初评

```text
教师触发 initial evaluation
  -> Go 聚合 submission + experiment + rubric + artifacts
  -> EvaluateRules 生成规则发现项和 rule 分数
  -> mode=rule_and_llm 时优先走 Python evaluate-submission
  -> Python 构造检索上下文并调用模型
  -> 如 Python 不可用且 Go LLM 已配置，则 Go 直连 LLM 兜底
  -> 结果写入 evaluation_results / metric_scores / llm_call_logs
```

输出约束：

- 模型输出必须是结构化 JSON。
- `metric_code` 必须属于当前 Rubric。
- 分数必须在 `0..max_score`。
- `evidence_refs` 必须来自允许证据集合。
- LLM 失败不会自动阻断教师复核，而是将评价标记为需要人工复核。

### 8.6 教师复核与发布

```text
读取提交详情 + 最新初评
  -> 保存教师复核草稿
  -> 逐指标确认/调整最终分
  -> 发布最终评价
  -> 学生查看已发布评价和报告
```

教师复核是最终成绩来源。AI 初评只提供建议，不直接发布。

### 8.7 报表与导出

```text
个人报告
实验统计
课程统计
  -> 生成 HTML / CSV / XLSX / PDF
  -> 写入 LocalStore reports/
  -> report_exports 记录 hash、大小、状态
  -> 下载时重新校验权限
```

报表相关实现集中在 `internal/teaching/report_service.go` 和 `internal/teaching/report_binary.go`。

## 9. 安全边界

当前安全基线：

- 用户密码使用 bcrypt hash。
- session token 只保存 hash。
- session cookie 默认 httpOnly，生产环境默认 secure。
- 修改类请求执行 CSRF 校验。
- 教师访问按课程授权判断。
- 学生访问限定为自己的提交和已发布评价。
- 上传文件使用服务端存储 key，防止路径穿越。
- LLM 输入只传必要证据摘要、规则和结构化上下文。
- LLM 输出必须校验 schema，失败时不得自动发布。
- 审计覆盖用户管理、课程授权、上传、初评、复核、发布和报表导出等关键动作。

详细要求见 `docs/SECURITY.md` 和 `docs/AUTH_SESSIONS.md`。

## 10. 构建、运行与验证

本地启动 Go 主服务：

```bash
go run ./cmd/server
```

执行升级迁移：

```bash
go run ./cmd/upgrade up
```

前端开发：

```bash
npm ci --prefix web
npm run dev --prefix web
```

前端构建：

```bash
npm run build --prefix web
```

构建内嵌前端的 Go 主服务：

```bash
go build -tags webui ./cmd/server
```

启动 Python AI Worker：

```bash
cd python-ai-gateway
python -m venv .venv
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

常用验证：

```bash
go test ./...
npm run build --prefix web
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build -tags webui ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/upgrade
```

## 11. 扩展建议

新增能力时建议按以下边界放置代码：

- 新增 HTTP 路由：优先放入 `internal/api` 或由领域包提供 `RegisterRoutes`。
- 新增教学业务：放入 `internal/teaching`，先定义领域输入/输出和 Repository 接口，再补 PostgreSQL / SQLite 实现。
- 新增认证能力：放入 `internal/authn`，避免散落在业务 handler。
- 新增模型调用能力：Go 编排放 `internal/llm` 或 `internal/aigateway`；复杂解析/RAG/模型适配优先放 Python AI Worker。
- 新增持久化表：同时新增 PostgreSQL 和 SQLite 迁移，不修改已发布迁移文件。
- 新增导出格式：扩展 `ReportFormat`、渲染函数、迁移约束和前端调用。
- 新增部署向导工具：放入 `internal/assistant`，保持“先确认、再执行、可审计”的工具调用模式。

需要特别避免：

- 把最终成绩发布逻辑放到 Python 或 LLM 侧。
- 在 API 进程内执行学生提交代码。
- 绕过 Repository 接口直接从 handler 写数据库。
- 在已发布 Rubric 或已发布教师评价上做普通更新。
- 把生产密钥、真实学生数据或导出报表提交到仓库。

## 12. 相关文档索引

- `README.md`：项目概览和快速启动。
- `PLAN.md`：交付计划和剩余门槛。
- `docs/TEACHING_DOMAIN.md`：教学域接口与数据模型。
- `docs/SUBMISSION_UPLOAD.md`：提交上传与解析。
- `docs/VERIFICATION_EVALUATION.md`：核查和初评。
- `docs/TEACHER_REVIEW.md`：教师复核与发布。
- `docs/REPORT_EXPORTS.md`：报表导出。
- `docs/PYTHON_AI_MIDDLEWARE.md`：Python AI Worker 当前实现。
- `docs/SINGLE_BINARY_RUNTIME.md`：单二进制、SQLite/PostgreSQL 运行方案。
- `docs/DEPLOY_KYLIN.md`：银河麒麟部署。
- `docs/SECURITY.md`：安全与合规基线。
- `docs/UAT_CHECKLIST.md`：UAT 操作清单。
