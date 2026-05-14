# loong64-b1-go 当前计划

## 1. 项目目标

本项目要交付的是一个面向高校 / 职教软件实训课程的教学评价系统，覆盖：

- 学生提交成果
- 系统解析材料
- 规则核查
- AI 初评
- 教师复核与发布
- 报表统计与导出

固定部署目标仍然是：

- `LoongArch + 银河麒麟高级服务器版`

当前项目已经不再适合继续用“纯 Go 自己硬扛全部 AI 能力”的思路推进。  
现在的正确定位是：

- `Go` 负责业务主系统
- `Python` 负责推理微服务

Python 的引入不是为了替代 Go，而是为了补齐 Go 在 AI 生态上的客观短板，尤其是：

- 本地大模型适配
- 文档解析生态
- 检索 / RAG
- 推理编排与结构化输出清洗

---

## 2. 当前架构

### 2.1 总体分层

系统现在分成 5 层：

1. `Web 前端`
2. `Go 主服务`
3. `Python 推理微服务`
4. `数据库 / 对象存储`
5. `模型服务`

### 2.2 Go 主服务职责

Go 是系统的主控层，负责：

- 登录、会话、权限、CSRF、安全边界
- 用户、课程、班级、实验、模板、提交、复核、发布
- 对象存储、数据库、审计日志、报表导出
- 对 Python 微服务的内部 HTTP 调用
- 最终业务真相源的持久化

一句话：

- `Go 负责系统状态和业务规则`

### 2.3 Python 推理微服务职责

Python 是内部 AI 能力层，负责：

- `parse-artifact`
- `evaluate-submission`
- `build-retrieval-index`
- `query-retrieval`
- 本地模型 / OpenAI-compatible 模型调用
- 检索上下文构建
- 结构化输出规范化与防御性校验

一句话：

- `Python 负责 AI-heavy 能力，不负责业务状态`

### 2.4 数据层职责

数据库保存：

- 用户与教学关系
- 提交记录
- 评价结果
- 教师复核结果
- 导出记录
- 审计日志
- LLM 调用日志

对象存储保存：

- 上传附件
- 解析中间产物
- 报表导出文件

### 2.5 模型层职责

模型层通过 OpenAI-compatible 接口接入，可来自：

- 本地大模型服务
- 校内部署模型网关
- 云端模型网关

这里的关键原则是：

- `业务模块不能直接绑定具体模型 SDK`
- `模型接入优先放在 Python 推理微服务内部`

---

## 3. 当前代码结构

### 3.1 Go 侧

- `cmd/`
  - `server`
  - `migrate`
- `internal/api`
  - REST API / middleware / handler
- `internal/authn`
  - 登录 / 会话 / actor 解析
- `internal/teaching`
  - 教学域主流程
- `internal/aigateway`
  - Go -> Python 微服务 client
- `internal/storage`
  - ObjectStore
- `internal/jobs`
  - 异步任务模型

### 3.2 Python 侧

- `python-ai-gateway/ai_gateway/app.py`
  - FastAPI 入口
- `python-ai-gateway/ai_gateway/parser.py`
  - 文档/附件解析
- `python-ai-gateway/ai_gateway/evaluator.py`
  - 评分请求组装、模型调用、结构化输出
- `python-ai-gateway/ai_gateway/retrieval.py`
  - 检索索引与查询
- `python-ai-gateway/ai_gateway/models.py`
  - 请求 / 响应模型

### 3.3 当前内部接口

- `GET /health/live`
- `GET /health/ready`
- `POST /internal/parse-artifact`
- `POST /internal/evaluate-submission`
- `POST /internal/build-retrieval-index`
- `POST /internal/query-retrieval`

---

## 4. 当前已完成

### 4.1 教学主链路

已经打通：

- 用户 / 课程 / 班级 / 实验 / 模板
- 学生提交与附件上传
- 教师触发初评
- 教师复核与发布
- 学生查看已发布结果
- HTML / CSV / XLSX / PDF 报表导出

### 4.2 Python 微服务基础能力

已经具备：

- Python 微服务骨架
- Go 侧健康检查接入
- 附件解析接线
- 初评接线
- 检索接口从 stub 变成真实实现
- parser metadata 的 `sections / evidence` 进入检索上下文
- 更细粒度的 evidence refs：
  - `artifact:<id>#section:n`
  - `artifact:<id>#evidence:n`

### 4.3 当前仍缺的不是“能不能跑”

真正还缺的是：

- AI 过程的可审计性更强
- 教师复核界面对 AI 证据的可见性更强
- Python 微服务部署方式正式纳入交付文档和 systemd 资产
- 本地模型 / embedding / 持久化检索后端路线明确化

---

## 5. 核心边界

这是当前项目必须坚持的边界：

### 5.1 Go 不负责的事

Go 不应该继续硬做这些事：

- 本地模型 SDK 适配
- 文档解析生态整合
- 检索 / RAG 主逻辑
- 模型输出清洗与编排

### 5.2 Python 不负责的事

Python 不应该负责：

- 用户权限
- 最终发布决策
- 教学业务主状态
- 直接写主业务数据库

### 5.3 统一原则

- `Go 管业务`
- `Python 管推理`
- `数据库只以 Go 为真相源`

---

## 6. 当前推荐部署形态

推荐把部署分成两种形态。

### 6.1 开发 / 调试

开发机上运行：

- `Go 主服务`
- `Python 推理微服务`
- `SQLite 或 PostgreSQL`
- `本地 / 远端模型服务`

### 6.2 生产 / 试点

银河麒麟目标机上建议运行：

- `loong64-b1-go.service`
- `python-ai-gateway.service`
- `PostgreSQL 或默认 SQLite`
- `本地模型服务或校内模型网关`

也就是说，部署说明必须从“只部署一个 Go 服务”升级成：

- `Go + Python 双服务部署`

---

## 7. 接下来要做的事

下面是基于当前状态的真正路线，不再按早期“从零开始搭框架”的视角写。

### 7.1 目标 A：文档与部署收口

目标：

- 把 `README.md`
- `PLAN.md`
- `docs/PYTHON_AI_MIDDLEWARE.md`
- `docs/DEPLOY_KYLIN.md`
- `deploy/kylin/systemd`

全部改到“Go 主服务 + Python 推理微服务”的真实架构上

交付：

- README 明确当前架构、调试方式、Python 本地运行方式
- 部署文档明确 Go / Python 双服务部署
- systemd 模板纳入 Python 微服务
- 环境变量模板纳入 `AI_GATEWAY_*` 与 `AI_GATEWAY_LLM_*`

验收：

- 新同学只看 README 和部署文档，就能理解系统分层
- 目标机部署说明不再遗漏 Python 微服务

### 7.2 目标 B：AI 过程审计可见

目标：

- 不只保存一个“模型输出结果”
- 还要把检索命中、引用摘要、命中数等信息回流成可审计数据

交付：

- `llm_call_logs` 输出中保留 retrieval context summary
- Go 侧测试覆盖“Python 返回的 retrieval 审计数据被持久化”

验收：

- 教师或管理员能在后端记录中追溯 AI 初评用了哪些主要证据

### 7.3 目标 C：教师复核可解释性

目标：

- 教师不是只看一个建议分
- 而是直接看到 AI 引用了哪些 section / evidence

交付：

- 教师初评详情 API 返回更清楚的 evidence refs / retrieval summary
- 前端复核页展示 AI 引用证据

验收：

- 教师复核时能直接判断“AI 为什么这么打”

### 7.4 目标 D：Python 检索与模型能力升级

目标：

- 给 Python 微服务明确一条长期可演进路线

优先顺序：

1. 当前有界内存检索继续稳定
2. 增加持久化检索后端替换点
3. 增加 embedding / 向量检索能力
4. 增加本地模型 provider 适配
5. 多 provider 路由与回退策略

验收：

- 不改变 Go 公共业务 API 的前提下，可以替换 Python 内部检索/模型实现

### 7.5 目标 E：LoongArch 试点交付

目标：

- 让现在这套双服务体系在目标机上可部署、可验证、可留档

交付：

- systemd 双服务
- 运行前检查
- 目标机验证记录
- UAT 跑单

验收：

- LoongArch + 银河麒麟可完成整条教学闭环

---

## 8. 当前优先级

按优先级排序：

1. `文档和部署说明改成双服务架构`
2. `检索上下文审计回流`
3. `教师复核可见 retrieval / evidence`
4. `Python 本地模型与检索后端升级`
5. `LoongArch 试点部署与 UAT`

---

## 9. LoongArch 约束

每次新增能力都要看这几件事：

- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`
- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate`
- Python 依赖是否可在目标机安装
- 是否强依赖 x86-only 组件
- 是否增加难以在银河麒麟维护的运行时

当前架构下的原则是：

- Go 继续尽量纯 Go
- Python 允许承接 AI 生态复杂度
- 但 Python 部署必须保持简单、可脚本化、可 systemd 管理

---

## 10. 风险与处理

### 10.1 Go / Python 边界漂移

风险：

- 业务状态慢慢跑进 Python

处理：

- Python 只返回结构化结果
- 主业务数据库只由 Go 写入

### 10.2 Python 依赖过重

风险：

- 目标机安装复杂

处理：

- 先走最小依赖方案
- 每次引入新依赖都写入部署文档

### 10.3 检索与模型能力升级过早

风险：

- 系统复杂度先于业务收益增长

处理：

- 先把审计与教师可见性做好
- 再做 embedding / 向量库

### 10.4 文档与实现脱节

风险：

- 团队理解混乱

处理：

- 任何架构变化都同步改 `PLAN.md`、`README.md`、部署文档

---

## 11. 当前结论

基于现在的情况，这个项目的正确表述应该是：

- `Go 是教学业务主系统`
- `Python 是推理微服务`
- `模型调用、本地模型适配、检索/RAG、文档解析增强优先放到 Python`
- `Go 继续负责权限、状态、复核、发布、审计、报表`

后面的工作不应该再模糊这条边界，而应该持续把它做实、做清楚、做可部署。
