# AGENT.md

本文件固定 `loong64-b1-go` 项目中所有 agent 和开发者的协作流程。任何会修改仓库内容的操作，都必须通过 GitHub MCP 在远端仓库形成 commit；不得只在本地修改后结束。

## 1. 固定项目身份

- GitHub Owner：`kenichiLyon`
- GitHub Repo：`loong64-b1-go`
- 远端地址：`https://github.com/kenichiLyon/loong64-b1-go`
- 项目定位：基于大模型技术的软件实训教学结果检查评价与报表系统。
- 技术路线：Go 后端 + PC Web 前端 + PostgreSQL + OpenAI-compatible LLM Gateway。
- 部署目标：自主指令系统 LoongArch 架构 + 银河麒麟高级服务器版。
- 开发环境：amd64 Windows + Linux；所有变更必须兼顾跨平台调试和 LoongArch 最终部署。
- 禁止误操作：不得复用、推送或修改 `loongarch-b1` 等旧仓库作为本项目交付物。

## 2. 每轮启动检查

每轮工作开始前必须完成：

1. 确认当前任务目标、影响范围和最小可验证工作单元。
2. 使用 GitHub MCP 确认目标仓库为 `kenichiLyon/loong64-b1-go`。
3. 如果远端仓库不存在，先通过 GitHub MCP 创建私有仓库，再继续。
4. 读取远端最新文件内容，尤其是 `AGENT.md`、`PLAN.md`、`README.md` 和相关代码。
5. 检查本地工作区是否存在未理解的变更；发现非本轮变更时停止并询问。
6. 明确本轮需要运行的最小验证命令和 LoongArch 影响。

只读探索不需要 commit。一旦修改仓库内容，必须提交到 GitHub 远端。

## 3. GitHub MCP 版本控制规则

- 所有仓库变更必须通过 GitHub MCP 完成远端 commit。
- 优先使用 GitHub MCP 工具：`get_me`、`create_repository`、`list_branches`、`get_file_contents`、`create_branch`、`push_files`、`create_or_update_file`、`create_pull_request`。
- 更新已有文件前，必须先读取远端当前内容；使用需要 SHA 的 MCP 方法时必须传入正确 SHA。
- 禁止只本地修改、只本地 commit、或绕过 MCP 直接把最终变更推到远端。
- 禁止 force push、删除远端分支、改写历史、`git reset --hard`、覆盖未读取的远端文件。
- 每个逻辑工作单元至少形成一个远端 commit；不要把无关功能混入同一提交。
- 每次完成一个工作单元后，必须把本地已完成改动通过 GitHub MCP 形成新的远端 commit；不得把已完成但未提交的改动长期留在工作区。
- 每次仓库变更提交都必须走 Pull Request 流程，等待 CI/CD、linter 和 SourceryAI review，并在 PR 中与 SourceryAI 的审查意见互动；若确认评论无实质性问题，可回复说明理由以说服 SourceryAI 后合并。
- GitHub MCP、网络或权限异常导致无法提交时，必须停止后续开发并报告阻塞原因。
- 每次最终回复必须列出：仓库、分支、commit SHA、变更文件、验证命令和验证结果。

### 分支策略

- `main`：稳定主分支，保存可运行的阶段性成果。
- 阶段 0 的治理初始化历史例外已结束；后续不得直接提交到 `main`。
- 所有阶段默认从 `main` 创建 `feature/<scope>`、`fix/<scope>` 或 `docs/<scope>` 分支。
- 所有变更必须创建 PR；小型文档修正也不得直接提交到 `main`。

### Commit Message 规范

使用 Conventional Commits：

- `chore(repo): bootstrap go solution governance`
- `docs(plan): update development roadmap`
- `feat(api): add submission upload workflow`
- `feat(llm): add openai compatible gateway`
- `test(eval): cover rubric weighting`
- `fix(report): handle failed pdf export`

禁止使用 `update`、`misc`、`wip` 作为最终提交说明。

## 4. 工作单元执行流程

1. 读取相关远端文件、代码、文档和测试。
2. 明确本次变更的业务目标、边界、验收方式和 LoongArch 风险。
3. 在本地完成最小必要修改，避免无关重构。
4. 执行最小验证：`gofmt`、`go test ./...`、必要 API/前端/导出检查。
5. 对 Go 后端变更执行 LoongArch 门槛检查：`GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`。
6. 提交前扫描敏感信息，不得提交密钥、Token、真实学生数据、生产文件或模型 API Key。
7. 通过 GitHub MCP 把本轮文件作为远端 commit 提交到工作分支。
8. 通过 GitHub MCP 创建 PR，并等待 CI/CD、linter、SourceryAI review 和必要人工检查完成。
9. 对 SourceryAI 评论逐条处理：有实质问题则修复并追加 commit；无实质问题则在 PR 中回复说明设计理由、风险评估和验证结果。
10. 所有阻塞检查通过且 SourceryAI 互动完成后，方可合并 PR。
11. 工作单元结束前必须执行一次本地状态检查；除明确保留给下一工作单元且已向用户说明的内容外，工作区必须恢复为 `git status` clean。

## 5. 技术约束

- Go 代码默认使用标准库和纯 Go 依赖；新增 CGO 依赖必须先写入 LoongArch 风险说明并获得确认。
- 后端采用模块化结构：`cmd/` 放入口，`internal/` 放业务实现，`api/` 放 OpenAPI，`migrations/` 放数据库迁移。
- 数据库使用 PostgreSQL；不要在 MVP 中同时支持多种关系数据库。
- 文件存储通过 `ObjectStore` 抽象；首版实现本地文件系统，预留 S3/MinIO 兼容层。
- LLM 调用必须通过统一 Gateway；支持云端或本地 OpenAI-compatible HTTP API，不允许业务模块直接绑定具体模型 SDK。
- 云端 LLM 默认只接收脱敏摘要、评分规则和必要证据片段；不得默认上传原始学生文件。
- 智能评价只能作为初评建议；最终成绩必须支持教师复核、改分、评语和发布留痕。
- 学生代码首版不在 API 服务进程中执行；如后续需要运行测试，必须放入独立 worker，限制用户、资源、超时和命令白名单。
- 报表以 HTML 视图作为规范展示源，Excel/PDF 作为异步导出产物，所有导出文件写入对象存储并记录生成条件。

## 6. 安全与合规规则

- 上传文件必须限制大小、数量、扩展名、MIME 类型和压缩包解压规模。
- 压缩包解析必须防止 Zip Slip、路径穿越、符号链接逃逸和超大解压。
- 学生姓名、学号、联系方式、源码仓库凭据等敏感信息在云端 LLM 调用前必须脱敏或最小化。
- Prompt 中必须固定系统评分规则优先级，防止学生提交内容覆盖评分规则。
- LLM 输出必须通过结构化 schema 校验；失败时进入人工复核或重试队列。
- 上传、解析、核查、LLM 调用、教师改分、结果发布、报表导出必须写审计日志。
- `.env`、API Key、数据库密码、私钥、真实学生提交物和生产导出报表不得提交仓库。

## 7. 测试与验收规则

- 后端新增业务逻辑必须配套单元测试或集成测试。
- 前端新增关键页面必须保证 TypeScript 检查和构建通过。
- 上传、解析、核查、评分、教师复核、导出属于核心链路，必须维护可重复样例。
- 阶段性验收至少运行：`go test ./...`、后端构建、LoongArch 交叉编译、核心 API 冒烟测试。
- 目标环境验收必须在 LoongArch + 银河麒麟高级服务器版完成 systemd 启动、健康检查、样例评价和报表导出。

## 8. LoongArch 与银河麒麟适配规则

- 所有新增依赖必须评估是否包含 native binary、是否需要 CGO、是否支持 `linux/loong64`。
- 默认交叉编译命令：`GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`。
- 交叉编译只是门槛，最终必须在 LoongArch 真机或等价环境运行 smoke test。
- 记录目标环境的 `uname -m`、银河麒麟版本、内核、glibc、Go、PostgreSQL、字体和 systemd 信息。
- PDF、Word、Excel、图片解析、OCR、本地模型服务等高风险能力必须提供降级策略和人工处理入口。
- 优先提供 systemd 非容器部署；容器/Podman 方案作为补充，不得作为唯一部署路径。

## 9. 完成标准

每个工作单元完成时必须满足：

- 变更已通过 GitHub MCP 提交到 `kenichiLyon/loong64-b1-go` 的工作分支，并通过 PR 流程合并。
- PR 已等待 CI/CD、linter 和 SourceryAI review；所有 SourceryAI 实质问题已修复，非实质问题已在 PR 中说明理由。
- 最终回复包含 commit SHA、文件清单、验证结果和未完成风险。
- 没有新增密钥、隐私数据或未评估的高风险依赖。
- `PLAN.md` 与实际范围保持一致；范围变化时必须同步更新。
- 本地工作区在交付时应为 clean，不遗留本轮已完成但未提交的改动。
