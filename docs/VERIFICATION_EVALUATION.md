# 规则核查与 LLM 初评

阶段 4 提供教师侧规则核查和可选 LLM 初评能力。当前接口提交初评任务，Go 侧通过数据库 job queue 记录状态、结果和失败原因；该阶段只产生“建议结果”，不写最终成绩，不向学生发布；教师复核、主观评分和发布流程放在阶段 5。

## API

- `POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial`
  - `mode=rule_only`：仅运行本地规则核查，默认值，适合离线 CI 和未配置模型环境。
  - `mode=rule_and_llm`：先运行规则核查，再通过 OpenAI-compatible 网关生成 LLM 指标建议分。
  - 返回初评 `job_id` 和当前任务状态。
- `GET /api/v1/teacher/evaluations/jobs/{jobID}`
  - 轮询初评任务状态；成功后返回本次初评结果。
- `GET /api/v1/teacher/submissions/{submissionID}/evaluations/latest`
  - 返回最新初评结果、规则发现和指标建议分。

只有 `teacher` 或 `admin` 可访问。普通教师必须已被授权到提交所属课程；学生不能触发或查看未发布初评。

## 数据表

- `evaluation_results`：一次初评运行的主记录，保存提交、任务、Rubric 版本、规则状态、LLM 状态、证据快照、摘要和待复核标记。
- `rule_check_findings`：完整性、解析状态、证据缺失、步骤覆盖、逻辑和 Prompt Injection 风险。
- `metric_scores`：按指标保存 `rule` 或 `llm` 来源的建议分、满分、置信度、理由和证据引用。
- `llm_call_logs`：只保存模型、Prompt 版本、输入 hash、结构化输出、状态、耗时和 token 计数；不保存 API Key 或原始文件内容。

## 规则核查范围

MVP 规则使用 `experiments.submission_spec`、`rubric_metrics`、`artifacts` 和 `extracted_contents`：

```json
{
  "required_artifacts": ["report", "code_archive", "screenshot"],
  "required_sections": ["实验环境", "步骤", "结果"],
  "required_steps": ["部署验证", "截图证明"],
  "keywords": ["LoongArch", "银河麒麟"]
}
```

当前规则包括：

- 必交成果类型缺失、无任何附件、代码包或 Git 链接缺失。
- 解析任务仍在排队/运行或解析失败。
- 文档/报告缺少文本摘要，导致教师需人工复核。
- 必要章节、步骤或关键词未在已有摘要中出现。
- 提交内容中出现 Prompt Injection 指令或疑似密钥字段。

上传后 `extracted_contents.status=queued` 不会阻塞规则核查；系统会将其标记为中等风险并降低规则建议分置信度。

## LLM 初评

LLM 网关使用标准库 `net/http` 调用 OpenAI-compatible `/v1/chat/completions`，配置项：

```bash
LLM_BASE_URL=http://127.0.0.1:8000/v1
LLM_MODEL=local-model
LLM_API_KEY=
LLM_TIMEOUT=30s
```

安全策略：

- 默认不发送原始文件，只发送脱敏证据快照、Rubric 指标、实训要求和允许引用的证据 ID。
- System prompt 明确学生提交是非可信证据，不得覆盖评分规则。
- LLM 必须返回 JSON；本地校验 `metric_code`、分数范围、置信度范围和证据引用。
- 输出无效、超时、网关未配置或模型错误时，结果进入 `needs_review`，不生成 LLM 指标分。

## LoongArch 兼容性

阶段 4 新增能力只使用 Go 标准库和已有 `pgx` 依赖，不引入 CGO、OCR、浏览器驱动、tokenizer 或本地模型运行时。LoongArch + 银河麒麟部署时，模型服务建议作为外部 OpenAI-compatible HTTP 网关配置；如使用 HTTPS 内网网关，应在系统层安装受信任 CA，不在代码中关闭 TLS 校验。

## 验证

```bash
gofmt -w cmd internal
go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/upgrade
```
