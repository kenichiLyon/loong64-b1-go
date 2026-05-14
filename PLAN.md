# loong64-b1-go 当前计划

## 1. 项目目标

本项目交付一个面向高校 / 职教软件实训课程的教学评价系统，覆盖：

- 学生提交成果
- 系统解析材料
- 规则核查
- AI 初评
- 教师复核与发布
- 个人报告、实验统计、课程统计与导出

固定部署目标：

- `LoongArch + 银河麒麟高级服务器版`

当前正式架构已经确定为：

- `Go 主服务`
- `Python AI Worker`

这里的核心判断已经不再变化：

- `Go 管业务`
- `Python 管异步 AI 任务`

## 2. 当前架构

### 2.1 运行单元

当前系统由 4 个核心运行单元组成：

1. `Go 主服务（内嵌 Web 前端 dist）`
2. `Python AI Worker`
3. `数据库 / 对象存储`
4. `模型服务`

### 2.2 Go 主服务职责

Go 是业务主控层，负责：

- 登录、会话、权限、CSRF 与安全边界
- 用户、课程、班级、模板、实验、提交、复核、发布
- 数据库和对象存储写入
- 审计日志、报表导出、对外 API
- Web 前端静态文件路由与 SPA fallback
- AI 任务入队、状态轮询、结果落库与并发限流

### 2.3 Python AI Worker 职责

Python 是 AI 任务执行层，负责：

- `parse-artifact`
- `evaluate-submission`
- `build-retrieval-index`
- `query-retrieval`
- 本地模型 / OpenAI-compatible 模型调用
- 检索上下文构建
- 结构化输出清洗与防御性校验
- 按 Go 下发的 job 执行 AI 重任务
- 遵守 worker 并发上限、超时、重试和任务状态协议

### 2.4 数据与模型层原则

- 主业务数据库只以 Go 为真相源
- Python 不直接写主业务数据库，结果必须经由 Go 定义的任务协议回写
- 模型接入优先放在 Python 微服务内部
- Go 业务模块不直接绑定具体模型 SDK
- 生产数据库主路径以 PostgreSQL 为准
- SQLite 仅作为本地调试、比赛演示和小规模试点入口

## 3. 当前实现状态

### 3.1 已完成

主线已经具备：

- 管理员 bootstrap、登录、改密、会话与 CSRF 防护
- 用户 / 班级 / 课程 / 选课 / 教师分配
- 模板 / 版本 / 实验任务创建与发布
- 学生提交、附件上传、Git 链接登记
- 规则核查、AI 初评、教师复核、最终发布
- 学生查看已发布评价
- HTML / CSV / XLSX / PDF 报表导出
- Python 微服务解析、同步初评、检索和细粒度 evidence refs
- 教师前端对 AI evidence 的多处可视化：
  - 详情页
  - 初评页
  - 复核页
  - 报告页

### 3.2 当前最重要的事实

当前代码主线已经接近“试点 MVP 可用”，但还不能把“代码能跑”直接等同于“项目可交付”。

阻塞真正交付的，主要是：

- 当前部署和调试体验仍被 nginx / split frontend 路径干扰，需要收口为 appembed 单入口
- 数据库访问层手写 SQL 和扫描代码过重，需要引入 sqlc 收敛生产主路径
- AI 初评仍依赖同步 HTTP 调用，不适合多学生并发场景
- UAT 记录还没有正式完成留档
- LoongArch / 银河麒麟目标机部署验收记录还没有正式完成
- 交付文档曾出现重复与失真，需要持续保持一致
- 需要把安全 / 会话 / 导出 / 部署验证结果沉淀成可交付材料

## 4. 交付标准

以下条件全部满足时，视为达到“可交付”：

1. 前端闭环完成：
   - bootstrap
   - 管理搭建
   - 教学配置
   - 学生提交
   - 教师初评 / 复核 / 发布
   - 学生查看已发布结果
2. 导出闭环完成：
   - 个人报告
   - 实验统计
   - 课程统计
   - HTML / CSV / XLSX / PDF
3. 代码验证完成：
   - `go test ./...`
   - `npm run build --prefix web`
   - `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`
   - `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate`
4. UAT 已实际执行并留痕
5. LoongArch / 银河麒麟部署验收已实际执行并留痕
6. README / PLAN / 部署文档 / Python 微服务文档一致

## 5. 当前剩余工作

### 5.1 A 类：架构收口与重构阻塞项

这些项目优先级最高，完成后才能严肃说“适合比赛展示和校园级演进”：

1. `彻底移除 nginx 默认路径`
   - 放弃 nginx 作为默认部署和调试入口
   - 使用 `go:embed` 嵌入 `web/dist`
   - Go 主服务注册静态文件路由和 SPA fallback
   - `/api`、`/health` 等保留为后端接口
   - 删除 nginx 配置、部署脚本、发布包、文档中的默认 nginx 依赖
   - nginx 相关内容必须删干净；如未来确需反代，只能作为单独 ADR 重新引入
2. `引入 sqlc 收敛数据库访问层`
   - 生产主路径优先支持 PostgreSQL + `pgx/v5`
   - 使用 sqlc 生成查询、参数和扫描代码
   - 保留显式事务边界，避免 ORM 式隐式行为
   - 设计查询时必须考虑多用户并发、锁等待、事务粒度、幂等和冲突处理
   - SQLite 定位为本地调试 / 比赛演示模式，不承诺承担校园级正式并发
3. `Python gateway 升级为异步 AI Worker`
   - 不再依赖同步 HTTP 请求完成初评
   - Go 创建 job，前端通过状态轮询查看进度和结果
   - Python worker 拉取或接收任务并异步执行
   - 初评、解析、检索等 AI 重任务必须进入 job queue
   - worker 必须支持并发限流、超时、重试、失败状态和可观测日志
   - 多学生并发时，任务上下文以 `job_id`、`submission_id`、`actor_id`、`conversation_id` 等显式字段隔离

### 5.2 B 类：交付验证项

这些项目完成后才能严肃说“可交付”：

1. `UAT 执行与留档`
2. `银河麒麟目标机部署验收与留档`
3. `安全与会话验证留档`
4. `交付文档一致性收口`

### 5.3 C 类：AI 可解释性增强

这些工作不一定阻塞 MVP 交付，但能显著改善试点体验：

1. 教师端 evidence 可见性增强
2. 报告页 AI 过程预览增强
3. 审计日志中 retrieval / evidence summary 更完整

### 5.4 D 类：后续演进项

这些不属于当前交付门槛：

1. 持久化检索后端
2. embedding / 向量检索
3. 本地模型 provider 扩展
4. OCR 深化
5. 更复杂的报表模板与图表

## 6. 下一步执行顺序

当前建议严格按下面顺序推进：

1. 收口 Web 部署入口：
   - 删除 nginx 默认路径
   - 确认 `web/dist` embed 到 Go 二进制
   - 注册 Go 静态文件路由与 SPA fallback
   - 更新 README / 部署文档 / 发布脚本
2. 引入 sqlc：
   - 先从较小模块试点
   - 再迁移 PostgreSQL 主路径的高频查询
   - 保持事务、锁和并发语义可审计
3. 将 Python gateway 改造为异步 AI Worker：
   - 设计 job 状态机
   - 设计 Go 与 Python 的任务协议
   - 接入状态轮询和并发限流
   - 把初评从同步 HTTP 改成异步任务
4. 收口并保持交付文档一致：
   - `README.md`
   - `PLAN.md`
   - `docs/DEPLOY_KYLIN.md`
   - `docs/PYTHON_AI_MIDDLEWARE.md`
5. 正式执行并填写：
   - `docs/UAT_CHECKLIST.md`
   - `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
6. 在目标环境完成双服务部署验证：
   - `loong64-b1-go.service`
   - `python-ai-worker.service`
7. 只对 UAT / 部署中暴露出的真实问题做修复
8. 最后整理交付包、验收记录和已知问题

## 7. LoongArch 约束

每轮新增能力都要过这几个门槛：

- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`
- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate`
- 新依赖是否支持 `linux/loong64`
- Python 依赖是否可在目标机安装
- 是否引入难以在银河麒麟维护的运行时

当前基本原则：

- Go 尽量纯 Go
- Python 承接 AI 生态复杂度
- Python 部署必须保持简单、可脚本化、可被 systemd 管理
- sqlc 生成代码必须确认支持 `linux/loong64`
- 删除 nginx 后，80 端口、TLS 或统一反代需求必须另行评估，不能回到默认依赖 nginx

## 8. 风险

### 8.1 文档继续漂移

风险：

- 团队和交付方对当前架构理解不一致

处理：

- 任何架构变化都同步修改 README / PLAN / 部署文档

### 8.2 nginx 残留导致交付路径分裂

风险：

- 本地调试、比赛演示和目标机部署继续出现多套入口

处理：

- 默认只保留 Go 单入口和 appembed 路线
- 删除 nginx 配置、包脚本和文档默认引用
- 后续如需反代，必须作为独立部署选项重新评审

### 8.3 sqlc 迁移扩大改动面

风险：

- 一次性迁移所有 repository 导致回归面过大

处理：

- 先小模块试点，再迁移主路径
- 每一步保留并发、事务和锁语义说明
- 迁移期间不改变业务行为

### 8.4 同步 AI 初评压垮并发

风险：

- 多学生同时提交或教师批量初评时，请求堆积、超时或模型服务过载

处理：

- 初评、解析和检索重任务全部走 job queue
- 前端轮询任务状态
- Python worker 设置并发上限、超时、重试和失败记录

### 8.5 Go / Python 边界漂移

风险：

- 业务状态逐渐跑进 Python

处理：

- Python 只返回结构化结果
- 主业务数据库只由 Go 写入

### 8.6 Python 依赖过重

风险：

- 目标机安装和维护复杂

处理：

- 先走最小依赖方案
- 依赖变更必须同步写进部署文档

### 8.7 过早优化检索 / 模型能力

风险：

- 系统复杂度先于交付收益增长

处理：

- 先把 UAT、部署、留档完成
- 再做向量库、本地模型 provider 等升级

## 9. 当前结论

当前项目的正确表述是：

- `Go 是教学业务主系统`
- `Web 前端默认内嵌在 Go 主服务中，不再依赖 nginx`
- `PostgreSQL 是校园级生产主数据库，sqlc 是生产主路径的数据访问收口方向`
- `Python 是异步 AI Worker，不再把同步 HTTP 初评作为正式并发方案`
- `当前主线已接近试点 MVP 可交付`
- `真正的剩余工作集中在 Web 入口收口、数据库访问层收口、AI 任务异步化、UAT、部署验收与交付留档`

后续工作不应再模糊这条边界，也不应再把“功能更多”误判成“更接近交付”。  
真正接近交付的标准，是：

- 功能闭环稳定
- 文档一致
- UAT 留痕
- 目标机可部署
- 验收材料齐全
