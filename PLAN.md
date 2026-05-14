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
- `Python 推理微服务`

这里的核心判断已经不再变化：

- `Go 管业务`
- `Python 管推理`

## 2. 当前架构

### 2.1 运行单元

当前系统由 5 个核心运行单元组成：

1. `Web 前端`
2. `Go 主服务`
3. `Python 推理微服务`
4. `数据库 / 对象存储`
5. `模型服务`

### 2.2 Go 主服务职责

Go 是业务主控层，负责：

- 登录、会话、权限、CSRF 与安全边界
- 用户、课程、班级、模板、实验、提交、复核、发布
- 数据库和对象存储写入
- 审计日志、报表导出、对外 API
- 对 Python 微服务的内部 HTTP 调用

### 2.3 Python 推理微服务职责

Python 是 AI 能力层，负责：

- `parse-artifact`
- `evaluate-submission`
- `build-retrieval-index`
- `query-retrieval`
- 本地模型 / OpenAI-compatible 模型调用
- 检索上下文构建
- 结构化输出清洗与防御性校验

### 2.4 数据与模型层原则

- 主业务数据库只以 Go 为真相源
- Python 不直接写主业务数据库
- 模型接入优先放在 Python 微服务内部
- Go 业务模块不直接绑定具体模型 SDK

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
- Python 微服务解析、初评、检索和细粒度 evidence refs
- 教师前端对 AI evidence 的多处可视化：
  - 详情页
  - 初评页
  - 复核页
  - 报告页

### 3.2 当前最重要的事实

当前代码主线已经接近“试点 MVP 可用”，但还不能把“代码能跑”直接等同于“项目可交付”。

阻塞真正交付的，主要是：

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

### 5.1 A 类：交付阻塞项

这些项目优先级最高，完成后才能严肃说“可交付”：

1. `UAT 执行与留档`
2. `银河麒麟目标机部署验收与留档`
3. `安全与会话验证留档`
4. `交付文档一致性收口`

### 5.2 B 类：AI 可解释性增强

这些工作不一定阻塞 MVP 交付，但能显著改善试点体验：

1. 教师端 evidence 可见性增强
2. 报告页 AI 过程预览增强
3. 审计日志中 retrieval / evidence summary 更完整

### 5.3 C 类：后续演进项

这些不属于当前交付门槛：

1. 持久化检索后端
2. embedding / 向量检索
3. 本地模型 provider 扩展
4. OCR 深化
5. 更复杂的报表模板与图表

## 6. 下一步执行顺序

当前建议严格按下面顺序推进：

1. 收口并保持交付文档一致：
   - `README.md`
   - `PLAN.md`
   - `docs/DEPLOY_KYLIN.md`
   - `docs/PYTHON_AI_MIDDLEWARE.md`
2. 正式执行并填写：
   - `docs/UAT_CHECKLIST.md`
   - `docs/STAGE7_DEPLOYMENT_VERIFICATION.md`
3. 在目标环境完成双服务部署验证：
   - `loong64-b1-go.service`
   - `python-ai-gateway.service`
4. 只对 UAT / 部署中暴露出的真实问题做修复
5. 最后整理交付包、验收记录和已知问题

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

## 8. 风险

### 8.1 文档继续漂移

风险：

- 团队和交付方对当前架构理解不一致

处理：

- 任何架构变化都同步修改 README / PLAN / 部署文档

### 8.2 Go / Python 边界漂移

风险：

- 业务状态逐渐跑进 Python

处理：

- Python 只返回结构化结果
- 主业务数据库只由 Go 写入

### 8.3 Python 依赖过重

风险：

- 目标机安装和维护复杂

处理：

- 先走最小依赖方案
- 依赖变更必须同步写进部署文档

### 8.4 过早优化检索 / 模型能力

风险：

- 系统复杂度先于交付收益增长

处理：

- 先把 UAT、部署、留档完成
- 再做向量库、本地模型 provider 等升级

## 9. 当前结论

当前项目的正确表述是：

- `Go 是教学业务主系统`
- `Python 是推理微服务`
- `当前主线已接近试点 MVP 可交付`
- `真正的剩余工作主要在 UAT、部署验收与交付留档`

后续工作不应再模糊这条边界，也不应再把“功能更多”误判成“更接近交付”。  
真正接近交付的标准，是：

- 功能闭环稳定
- 文档一致
- UAT 留痕
- 目标机可部署
- 验收材料齐全
