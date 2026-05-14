# loong64-b1-go

基于大模型技术的软件实训教学结果检查评价与报表系统。  
当前仓库面向 `LoongArch + 银河麒麟高级服务器版` 交付，采用 `Go 主服务 + Python 推理微服务` 的双服务架构。

## 目标

- 支持本地或云端 OpenAI-compatible 模型服务
- 提供 PC Web 工作台，覆盖管理员、教师、学生核心流程
- 支持报告、Word、PDF、截图、代码包、Git 链接等成果上传与解析
- 实现规则核查、AI 初评、教师复核与最终发布
- 生成个人报告、实验统计、课程统计，并支持导出

## 当前状态

当前主线已经具备试点 MVP 的核心闭环：

- bootstrap 初始化
- 登录 / 会话 / CSRF / 改密
- 管理员搭建用户、班级、课程、选课
- 教师搭建模板、版本、实验任务
- 学生提交成果
- 系统解析、规则核查、AI 初评
- 教师复核、发布
- 学生查看已发布结果
- HTML / CSV / XLSX / PDF 导出

当前离“正式可交付”还剩的重点不是大块新功能，而是：

- UAT 执行与留档
- 银河麒麟目标机部署验收与留档
- 交付文档一致性

## 当前架构

系统由 4 类核心组件组成：

1. `Web 前端`
2. `Go 主服务`
3. `Python 推理微服务`
4. `数据库 / 对象存储 / 模型服务`

职责边界：

- `Go 主服务`
  - 对外 API
  - 登录 / 会话 / 权限 / 审计
  - 教学业务流程
  - 数据库和对象存储写入
  - 报表与导出
- `Python 推理微服务`
  - `parse-artifact`
  - `evaluate-submission`
  - `build-retrieval-index`
  - `query-retrieval`
  - 本地模型 / OpenAI-compatible 模型调用
  - 检索上下文和结构化输出整理

一句话：

- `Go 管业务`
- `Python 管推理`

## 快速启动

启动 Go 主服务：

```bash
go run ./cmd/server
```

默认监听：

- `http://127.0.0.1:8080`

默认数据库模式是本地 SQLite：

```bash
RUNTIME_CONFIG_PATH=./config/runtime.json
DB_DRIVER=sqlite
SQLITE_PATH=./data/loong64-b1-go.db
AUTO_MIGRATE=true
```

### 启动 Python 推理微服务

Linux / macOS：

```bash
cd python-ai-gateway
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Windows PowerShell：

```powershell
cd python-ai-gateway
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Go 侧启用 Python 微服务时，至少配置：

```bash
AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_TIMEOUT=10s
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

## 前端开发

```bash
cd web
npm ci
npm run dev
npm run build
```

开发服务器默认代理 `/api` 和 `/health` 到 `http://127.0.0.1:8080`。

## 数据库迁移

SQLite：

```bash
DB_DRIVER=sqlite SQLITE_PATH=./data/loong64-b1-go.db go run ./cmd/migrate up
```

PostgreSQL：

```bash
DB_DRIVER=postgres DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable go run ./cmd/migrate up
```

## 验证

```bash
go test ./...
npm run build --prefix web
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate
```

## 认证与教学 API

主链路认证接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `PUT /api/v1/auth/password`
- `GET /api/v1/me`
- `PUT /api/v1/admin/users/{userID}/password`

教学域核心接口：

- 学生提交：`POST /api/v1/student/experiments/{experimentID}/submissions`
- 上传附件：`POST /api/v1/student/submissions/{submissionID}/artifacts`
- 登记 Git 链接：`POST /api/v1/student/submissions/{submissionID}/artifact-links`
- 触发初评：`POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial`
- 教师复核：`PUT /api/v1/teacher/submissions/{submissionID}/review`
- 发布结果：`POST /api/v1/teacher/submissions/{submissionID}/review/publish`
- 学生查看已发布评价：`GET /api/v1/student/submissions/{submissionID}/review`
- 报告导出：`POST /api/v1/teacher/.../report-exports`

## 推荐部署方式

试点 / 生产环境推荐部署为双服务：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

如果不启用 Python 微服务，Go 侧仍可保留部分回退能力；但当前推荐路径已经是双服务协作。

## 关键文档

- 计划与交付门槛：`PLAN.md`
- MVP 范围：`docs/MVP_DELIVERY.md`
- UAT 手册：`docs/UAT_CHECKLIST.md`
- 银河麒麟部署：`docs/DEPLOY_KYLIN.md`
- Python 微服务：`docs/PYTHON_AI_MIDDLEWARE.md`
- Stage 7 部署验证：`docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
- 容器次级交付：`docs/CONTAINER_RUNTIME.md`
- 默认 SQLite / PostgreSQL 运行方案：`docs/SINGLE_BINARY_RUNTIME.md`

## 目录

```text
.github/workflows/     CI/CD 与代码审核
cmd/                   Go 入口
internal/              后端内部模块
api/                   OpenAPI
migrations/            数据库迁移
web/                   PC Web 前端
python-ai-gateway/     Python 推理微服务
deploy/kylin/          银河麒麟部署脚本和 systemd 文件
docs/                  架构、安全、兼容性和交付文档
```

## 版本控制纪律

本项目遵循 `AGENT.md`：

- 仓库修改必须形成远端 commit
- 每个工作单元通过 PR 合并
- 最终交付时本地工作区必须 `git status` clean
