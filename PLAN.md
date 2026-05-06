# loong64-b1-go 开发计划

## 1. 摘要

本项目建设“基于大模型技术的软件实训教学结果检查评价与报表系统”，用于高校或职教软件实训课程的成果收集、自动解析、智能核查、多维评价、教师复核和报表导出。

固定部署目标为 LoongArch 架构 + 银河麒麟高级服务器版；开发阶段使用 amd64 Windows + Linux，但每个阶段都必须保留 LoongArch 交叉编译和目标环境验证任务。

首版交付形态为 PC Web B/S 系统：Go 后端提供 API、异步任务、文件解析、LLM Gateway 和报表导出；前端提供教师、学生、管理员工作台。大模型只作为初评与核查助手，最终成绩保留教师复核、主观评分和审计留痕。

## 2. 技术基线

- 后端：Go 1.24+，默认标准库优先，必要时引入纯 Go 依赖。
- API：REST + OpenAPI；后续可按需增加 SSE/WebSocket 展示任务进度。
- 前端：Vue 3 + Vite + TypeScript，PC Web 优先，移动端做响应式适配，不开发原生 App。
- 数据库：PostgreSQL，保存结构化业务数据、评分结果、任务状态和审计日志。
- 文件存储：本地 ObjectStore 起步，目录化保存上传物、解析中间产物和报表文件，预留 MinIO/S3。
- 大模型：OpenAI-compatible HTTP Gateway，支持云端模型、本地模型或学校内网模型服务。
- 报表：HTML 报表视图 + Excel/PDF 异步导出，图表数据与导出条件可追溯。
- 部署：systemd 非容器部署为主，Podman/Docker 方案为辅。
- 版本控制：所有仓库变更必须通过 GitHub MCP commit 到 `kenichiLyon/loong64-b1-go`。

## 3. 总体架构

系统分为六条主链路：

1. 用户与教学管理：管理员维护账号、角色、课程、班级、选课和教师授权。
2. 实训任务与评价模板：教师创建实训要求、提交规范、评价指标、权重和评分规则版本。
3. 成果上传与解析：学生上传 Word、PDF、报告、截图、代码包或 Git 链接，系统保存文件并创建解析任务。
4. 智能核查与初评：规则引擎检查提交完整性、步骤覆盖、逻辑风险和格式问题，LLM Gateway 对脱敏证据进行初评。
5. 教师复核与发布：教师查看证据、AI 建议、规则扣分点，逐项改分、填写主观评价并发布结果。
6. 报表统计与导出：生成个人评价报告、班级统计、课程统计和常见问题分析，支持 Excel/PDF 导出。

## 4. 核心数据对象

- `users`：管理员、教师、学生账号，保存角色和状态。
- `courses`、`classes`、`enrollments`：课程、班级和选课关系。
- `experiments`：实训任务、提交要求、截止时间、可上传类型。
- `rubric_templates`、`rubric_metrics`：评价模板版本、指标、权重、评分说明。
- `submissions`：学生或小组提交记录，保存提交状态和当前有效版本。
- `artifacts`：上传文件或 Git 链接元信息，保存 hash、大小、MIME、对象存储 key。
- `extracted_contents`：文档文本、截图 OCR、代码结构、报告章节、关键证据片段。
- `rule_check_findings`：步骤缺失、逻辑漏洞、格式问题、风险标签和证据来源。
- `evaluation_results`、`metric_scores`：AI 初评、规则分、指标建议分、置信度和证据引用；教师分、最终分、评语和发布状态在阶段 5 引入。
- `llm_call_logs`：模型、Prompt 版本、输入摘要 hash、结构化输出、耗时、错误状态。
- `report_exports`：报表类型、筛选条件、文件格式、对象存储 key、生成状态。
- `audit_logs`：上传、解析、评价、改分、发布、导出等关键操作留痕。
- `jobs`：解析、核查、LLM 评价和报表导出等异步任务状态。

## 5. 接口与功能边界

### 5.1 用户侧 PC Web

- 学生：查看实训任务、上传成果、查看解析状态、查看已发布评价报告。
- 教师：创建任务、配置指标权重、查看提交进度、复核 AI 初评、发布成绩、导出报表。
- 管理员：维护用户、班级、课程、系统配置、模型配置和审计查询。

### 5.2 LLM Gateway

- 统一接口：`EvaluateSubmission(ctx, request) -> structured result`。
- Provider：OpenAI-compatible API，参数包括 base URL、model、API key、timeout、max tokens。
- 输入策略：默认发送脱敏摘要、评分规则、证据片段和任务要求，不发送原始文件。
- 输出策略：必须返回 JSON，包含指标建议分、理由、证据引用、风险标签、置信度。
- 失败策略：schema 校验失败、超时、限流或模型错误时进入重试或教师待复核。

### 5.3 文件解析

- Word/PDF：提取正文、标题层级、表格摘要和关键截图引用。
- 报告类文档：识别实验目的、环境、步骤、结果、问题分析、总结等章节。
- 截图：保存图片元信息，OCR 作为可选能力；目标环境不可用时保留人工查看入口。
- 代码包：只做静态结构检查、文件清单、语言识别、README/配置/测试文件存在性检查；MVP 不自动执行学生代码。
- Git 链接：先记录 URL 和提交信息；如需拉取，必须走独立 worker 和白名单策略。

### 5.4 报表导出

- 学生个人报告：最终分、指标分、AI 建议摘要、教师评语、主要证据和改进建议。
- 班级统计：分数分布、指标均值、提交完成率、常见问题 Top N、异常提交列表。
- 课程统计：跨班级对比、任务完成情况、模型初评与教师改分差异。
- Excel：结构化数据表、统计汇总、必要图表数据。
- PDF：面向归档和汇报，必须支持中文字体；图表以可验证方式嵌入或降级为表格。

## 6. 阶段计划与验收

### 阶段 0：仓库治理与初始骨架

目标：创建 `loong64-b1-go` 私有远端仓库，固化 AGENT 流程和完整计划，提交最小 Go 服务骨架。

交付：

- `AGENT.md` 固定 MCP 远端提交流程。
- `PLAN.md` 保存本开发计划。
- `README.md`、`.gitignore`、`.env.example`、`go.mod`。
- `cmd/server` 最小健康检查服务。
- `api/openapi.yaml` 初始健康检查接口说明。
- `docs/LOONGARCH_COMPATIBILITY.md`、`docs/SECURITY.md`。

验收：

- GitHub 远端仓库存在且初始文件已通过 MCP commit。
- `go test ./...` 通过。
- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server` 通过。

### 阶段 1：后端基础设施

目标：建立可配置、可迁移、可审计的 Go 后端基础。

交付：

- 配置加载、结构化日志、错误响应、请求 ID、中间件。
- PostgreSQL 连接池和迁移命令。
- 本地 ObjectStore 抽象和存储目录初始化。
- Job 状态模型和基础任务执行器。
- OpenAPI 初始规范和 API 冒烟测试。
- Auto Build 流水线生成 linux/amd64 与 linux/loong64 构建产物，Code Quality Review 流水线执行 linter 与 SourceryAI PR 审核，CD 流水线发布 Auto Build 产物到 GitHub Release。

验收：

- 健康检查区分 `live`、`ready`，ready 覆盖数据库和存储。
- 空数据库可一键迁移到最新 schema。
- PR 可自动执行 Go linter；配置 `SOURCERY_TOKEN` 后可执行 SourceryAI 差异代码审核。
- Auto Build 成功上传 artifact，CD 只发布 Auto Build 产物且不重新编译。
- Windows/Linux amd64 本地可运行，LoongArch 交叉编译通过。

### 阶段 1.5：部署骨架与本地数据库调试

目标：补齐银河麒麟 systemd 非容器部署骨架和开发环境 PostgreSQL 初始化脚本，降低后续阶段联调成本。

交付：

- `deploy/kylin/systemd` 提供 API 服务和迁移服务的 systemd 单元。
- `deploy/kylin/env` 提供生产环境变量模板，不包含真实密钥。
- `deploy/kylin/scripts` 提供安装单元和健康检查脚本。
- `scripts/dev` 提供 Windows PowerShell 与 Linux shell 的本地 PostgreSQL 初始化和启动脚本。
- `docs/DEPLOY_KYLIN.md` 与 `docs/LOCAL_POSTGRES.md` 记录部署与本地联调步骤。

验收：

- 本地脚本通过语法级检查，不提交真实密码或密钥。
- systemd 单元包含专用用户、最小写目录、环境文件和 restart 策略。
- 文档覆盖 release 产物、迁移、服务启动、健康检查和目标机验证记录。

### 阶段 2：用户、课程与评价模板

目标：完成教学管理和评分规则基础闭环。

交付：

- 用户、角色、课程、班级、选课关系。
- 实训任务 CRUD 和发布状态。
- 评价模板、指标、权重、版本化。
- 权重校验：使用整数基点 `weight_bps`，`strict_100` 模式总权重为 10000，`normalized` 模式按配置归一化。
- 已发布模板版本和指标不可变，实训任务绑定具体模板版本。
- 管理员和教师基础页面原型后续接入；本阶段优先完成后端 API 与 OpenAPI 契约。

验收：

- 教师可创建任务并绑定评价模板。
- 模板修改不影响历史评价结果。
- 后端单元测试覆盖权重校验、角色权限、模板所有者权限和版本绑定。
- `go test ./...`、`golangci-lint`、LoongArch server/migrate 交叉编译通过。

### 阶段 3：成果上传与解析

目标：支持多格式成果上传，形成可追溯解析证据。

交付：

- `submissions`、`artifacts`、`extracted_contents` 数据表和解析 Job 队列挂接。
- 学生创建提交、上传文件、登记 Git 链接；教师查看提交列表和详情。
- 上传接口、大小/类型/数量限制、SHA-256 计算和对象存储落盘。
- Word/PDF/报告/截图/代码包/Git 链接元信息解析；深度解析后续进入 worker。
- 解析任务状态流：queued、running、succeeded、failed。
- 压缩包安全检查：Zip Slip、路径穿越、符号链接、超大解压。
- 学生提交页面和教师提交详情页。

验收：

- 样例 PDF、文本报告、截图和代码包可上传并生成解析摘要或元数据。
- 非法类型、超大文件和恶意压缩包被拒绝并记录原因。
- 解析失败不会阻塞教师人工查看原始文件。
- `go test ./...`、`golangci-lint`、LoongArch server/migrate 交叉编译通过。

### 阶段 4：规则核查与 LLM 初评

目标：减少人工排查成本，并提供可复核的智能评价建议。

交付：

- `evaluation_results`、`rule_check_findings`、`metric_scores`、`llm_call_logs` 迁移表，保存初评运行、规则发现、指标建议分和 LLM 调用摘要。
- 教师同步触发 API：`POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial`；教师查询最新初评 API：`GET /api/v1/teacher/submissions/{submissionID}/evaluations/latest`。
- 规则引擎：提交完整性、步骤覆盖、文档结构、关键证据、解析状态、明显逻辑风险和 Prompt Injection 风险。
- LLM Gateway：标准库实现 OpenAI-compatible `/v1/chat/completions`，支持 base URL、model、API key、timeout。
- 脱敏摘要：学生隐私、联系方式、密钥、仓库凭据最小化；`llm_call_logs` 只保存输入 hash 和结构化输出。
- 结构化评分输出 schema：指标建议分、理由、证据引用、风险和置信度；本地校验 metric code、分数范围和证据引用。
- Mock/httptest LLM 测试，保证离线 CI 可验证。

验收：

- 同一提交可看到规则核查结果、AI 初评和证据来源。
- LLM 输出不合规时进入待人工复核，不覆盖最终成绩。
- Prompt injection 样例不能覆盖系统评分规则。
- `go test ./...`、`golangci-lint`、LoongArch server/migrate 交叉编译通过。

### 阶段 5：教师复核与成绩发布

目标：让智能初评变成可解释、可调整、可发布的教学评价结果。

交付：

- `teacher_reviews` 和 `teacher_metric_scores` 迁移表，保存教师复核草稿、逐指标最终分、来源、改分理由和发布状态。
- 教师复核 API：保存草稿、查看草稿/已发布结果、显式发布最终评价。
- 学生已发布评价 API：学生只能查看自己已发布的最终评价，未发布草稿不可见。
- 教师主观评分、改分理由、评语和最终确认。
- 发布后数据库触发器阻止复核主表和逐指标分被修改或删除。
- 所有保存和发布行为写审计日志。

验收：

- 教师可从 AI 初评分生成最终分，也可完全手动评分。
- 发布后结果不可被后台任务覆盖。
- 学生只能查看已发布最终评价。
- 改分差异可被审计查询和报表统计。
- `go test ./...`、`golangci-lint`、LoongArch server/migrate 交叉编译通过。

### 阶段 5.5：PC Web MVP

目标：在后端评价闭环稳定后，交付学生、教师、管理员可操作的 PC Web 最小界面。

交付：

- Vue 3 + Vite + TypeScript 前端骨架与 `npm run build` 验证。
- 学生端：任务查看、提交创建、成果上传、提交详情、已发布评价查看。
- 教师端：提交列表、解析/核查结果、AI 初评、复核草稿、发布成绩。
- 管理端：本阶段保留后端管理 API；完整管理工作台延后到阶段 6/7 补齐。
- 开发态使用 `X-Actor-ID` / `X-Actor-Roles` 模拟登录，生产认证后续接入。

验收：

- 前端可通过现有 API 完成上传、初评、教师复核、发布、学生查看的主流程演示。
- 桌面端可用，移动端基础响应式不破版。
- 前端构建和后端 CI 不互相阻塞；LoongArch 部署文档明确前端静态资源托管方式。
- 当前 MVP 页面已覆盖学生提交/上传、教师核查/复核/发布、学生查看已发布评价的演示链路。

### 阶段 6：统计报表与 Excel/PDF 导出

目标：满足课程评价、班级统计和归档汇报要求。

当前 Stage 6 采用“纯 Go、LoongArch 安全优先”的报表闭环：已先后交付个人报告、实验统计、课程跨实验统计、HTML/CSV 导出记录与下载；PDF 暂按降级策略记录为待配置，避免在未验证的 LoongArch + 银河麒麟环境中引入浏览器、LibreOffice headless 或 native PDF 依赖。

已交付：

- 学生个人评价报告 API：教师可查看草稿/发布上下文，学生只能查看自己已发布报告。
- 实验统计报表 API：提交数、已提交数、已发布评价数、分数分布、指标均值、附件解析状态和常见问题统计。
- 课程跨实验统计 API：按课程聚合实验数、提交数、已发布评价数、分数分布、指标均值和常见问题，并保留各实验子摘要。
- `report_exports` 迁移表：记录报表类型、范围、格式、状态、筛选条件、对象存储 key、SHA-256、大小和操作者。
- HTML 导出：作为规范归档源，写入对象存储 `reports/` 目录。
- CSV 导出：带 UTF-8 BOM，作为 Excel/WPS/LibreOffice 兼容的首版表格导出。
- PC Web 报表面板：支持个人报告预览、实验统计预览、课程统计预览、HTML/CSV 导出和 PDF 降级结果展示。

后续交付：

- 班级统计面板与跨班级对比，补齐班级维度完成率、异常提交和模型初评与教师改分差异。
- Excel XLSX 异步导出，包含明细表、汇总表和图表数据；首选纯 Go 依赖，失败降级 CSV。
- PDF 异步导出，需先选定 LoongArch 可验证的中文字体与渲染策略；图表不可用时降级为表格摘要。
- 后台异步 worker 重试、导出任务列表和管理员审计查询。

验收：

- 个人报告、实验统计和课程统计 API 可在教师/学生权限下正确返回或拒绝访问。
- HTML/CSV 导出文件写入对象存储，并记录 SHA-256、大小、筛选条件和操作者。
- CSV 可被 WPS/LibreOffice/Microsoft Excel 打开；PDF 在未配置 LoongArch 验证渲染器时必须给出明确降级记录。
- 50 份以上样例提交可生成班级统计和课程统计。
- PDF 在银河麒麟环境可生成、可打开、中文不乱码；图表不可用时自动降级为表格摘要。

### 阶段 7：LoongArch + 银河麒麟部署验证

目标：完成目标环境可运行、可维护、可升级验证。

当前切片已开始交付部署验证资产：主机环境采样脚本、部署前置检查脚本、整体验证脚本、Nginx 静态托管示例，以及数据库备份恢复与 LLM 配置示例文档。

交付：

- systemd 服务文件、环境变量模板、目录权限说明。
- PostgreSQL 初始化、迁移、备份恢复步骤。
- 前端构建产物托管方案。
- 本地/云端 LLM 配置示例。
- LoongArch 依赖验证记录和不可用能力降级说明。

验收：

- 在 LoongArch + 银河麒麟高级服务器版完成干净部署。
- 服务开机自启动，`/health` ready 通过。
- 可完成登录、任务创建、成果上传、解析、AI mock 评价、教师发布和报表导出。
- 记录目标机 `uname -m`、系统版本、Go、PostgreSQL、字体和部署命令。

### 阶段 7.5：单二进制运行与默认 SQLite

目标：把系统推进到“一个二进制即可提供 API + Web UI，默认不依赖外部数据库”的交付形态。

交付：

- 后端内嵌 `web/dist`，单二进制直接托管前端页面。
- 发布产物区分纯后端二进制和包含内嵌前端的完整二进制。
- 配置新增数据库驱动选择：`sqlite` / `postgres`。
- 默认 SQLite 运行模式，支持首次启动初始化。
- PostgreSQL 保留为生产模式数据库。
- 首次启动 / 管理员设置向导，用于选择数据库模式并写入本地运行时配置。

验收：

- 直接运行二进制即可打开 UI，不依赖额外 Web 服务器。
- 默认 SQLite 模式下可完成最小演示链路。
- PostgreSQL 模式保持现有主链路能力。
- 数据库切换通过初始化向导或管理员设置完成，并明确要求重启生效。

### 阶段 8：安全、性能、UAT 与发布

目标：完成试点验收和可交付发布包。

交付：

- 上传安全、越权访问、Prompt Injection、敏感信息泄露测试。
- 50-200 份样例提交批量试跑，记录解析和评价耗时。
- 用户手册、部署手册、管理员手册和演示脚本。
- 发布包、版本 tag、回滚说明和已知问题清单。

验收：

- 试点课程数据演练通过。
- 高危问题关闭，中低风险有明确缓解措施。
- 发布包可在新环境按文档复现部署。

## 7. LoongArch 专项检查清单

每次新增依赖或阶段验收都必须检查：

- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server` 是否通过。
- 是否引入 CGO；如引入，列出 LoongArch 原生依赖包和安装命令。
- 是否包含 x86-only 二进制、浏览器驱动、PDF 工具、OCR 工具或 Node 运行时依赖。
- PostgreSQL、字体、时区 `Asia/Shanghai`、UTF-8、文件权限和 systemd 用户是否正确。
- PDF/Word/Excel/图片解析库是否纯 Go 或可在 LoongArch 编译。
- 本地模型服务是否支持 LoongArch；不支持时必须保留云端或内网模型服务配置。
- 目标环境验证记录必须追加到 `docs/LOONGARCH_COMPATIBILITY.md`。

## 8. 测试策略

- 单元测试：权重校验、规则核查、脱敏、LLM schema 校验、文件类型检测。
- 集成测试：上传到解析、解析到评价、教师复核到发布、报表导出任务。
- 安全测试：越权访问、恶意压缩包、Prompt Injection、敏感信息扫描。
- 兼容测试：Windows amd64、Linux amd64、Linux loong64 交叉编译和目标机 smoke test。
- 报表测试：Excel 打开兼容性、PDF 中文字体、图表降级、导出条件审计。
- 性能测试：50、100、200 份提交批量解析和评价，记录 P50/P95 耗时。

## 9. 默认假设

- 首版只做 PC Web，不开发原生 App。
- 首版不在 API 服务进程内执行学生代码；代码包以静态分析为主。
- PostgreSQL 是唯一结构化数据库。
- LLM Provider 采用 OpenAI-compatible HTTP，不绑定单一厂商 SDK。
- 云端模型默认不接收原始学生文件，只接收脱敏摘要和必要证据。
- PDF 导出必须交付，但实现允许按 LoongArch 可用能力降级图表表现形式。
- 阶段 0 可直接提交到 `main`；后续功能开发默认使用 feature 分支。

## 10. 主要风险与缓解

- LoongArch 生态差异：坚持纯 Go 和无 CGO 默认，尽早做真机 smoke test。
- 文档解析依赖风险：优先选纯 Go 或外部可替换工具，失败时保留人工查看入口。
- PDF 中文和图表风险：HTML 作为规范展示源，PDF 模板单独验证字体和图表降级。
- LLM 误判风险：AI 只做初评，教师复核和证据链作为最终质量控制。
- 隐私合规风险：脱敏、最小化、审计和不保存密钥作为默认策略。
- 需求扩张风险：所有新增能力必须落入阶段计划，并通过 MCP commit 更新 `PLAN.md`。
