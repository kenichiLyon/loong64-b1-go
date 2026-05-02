# 成果上传与解析

阶段 3 交付提交记录、成果附件、Git 链接登记和解析任务队列骨架，为后续规则核查与 LLM 初评提供可追溯证据。

## 数据模型

- `submissions`：学生针对某个实训任务创建的提交记录，当前使用 `attempt_no=1`，后续可扩展多次提交。
- `artifacts`：附件或 Git 链接元信息，保存类型、原始文件名、MIME、大小、SHA-256、对象存储 key、来源 URL 和解析元数据。
- `extracted_contents`：解析结果状态、文本摘要、结构化元数据和失败原因。
- `jobs`：上传文件会创建 `submission_artifact_parse` 队列任务，后续 worker 可继续执行 Word/PDF/OCR/代码静态分析。
- `audit_logs`：创建提交、上传附件、登记链接都会写入审计日志。

## API 流程

1. 学生创建提交：`POST /api/v1/student/experiments/{experimentID}/submissions`
2. 学生上传文件：`POST /api/v1/student/submissions/{submissionID}/artifacts`
3. 学生登记 Git 链接：`POST /api/v1/student/submissions/{submissionID}/artifact-links`
4. 教师查看列表：`GET /api/v1/teacher/experiments/{experimentID}/submissions`
5. 教师或学生查看详情：`GET /api/v1/teacher/submissions/{submissionID}` 或 `GET /api/v1/student/submissions/{submissionID}`

开发环境仍使用临时操作者请求头：

```bash
X-Actor-ID: student-1
X-Actor-Roles: student
```

上传示例：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/student/submissions/sub_xxx/artifacts \
  -H "X-Actor-ID: student-1" \
  -H "X-Actor-Roles: student" \
  -F "artifact_kind=report" \
  -F "file=@report.md"
```

Git 链接示例：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/student/submissions/sub_xxx/artifact-links \
  -H "X-Actor-ID: student-1" \
  -H "X-Actor-Roles: student" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.edu/group/repo.git","commit_sha":"abcdef1"}'
```

## 上传限制

- `MAX_UPLOAD_BYTES`：单文件最大字节数，默认 `52428800`。
- `MAX_ARTIFACTS_PER_SUBMISSION`：单个提交最多附件数，默认 `20`。
- 支持扩展名：`.doc`、`.docx`、`.pdf`、`.txt`、`.md`、`.png`、`.jpg`、`.jpeg`、`.zip`。
- 对图片、PDF、ZIP 类文件执行内容嗅探，明显 MIME 不匹配会拒绝。
- 对 `.zip` 和 `.docx` 执行容器安全检查，包括路径穿越、反斜杠/盘符、符号链接、文件数量和解压总量限制。
- Git 链接只登记元数据，不在 API 进程内拉取仓库；后续如需拉取必须进入独立 worker 和白名单策略。

## 解析边界

当前阶段采用目标环境安全优先的元数据解析：

- 文本和 Markdown：提取有限长度文本摘要。
- 截图：解析图片宽高。
- ZIP 代码包：记录安全文件清单、扩展名分布和总解压大小。
- DOCX：按 ZIP 容器做安全检查并记录容器摘要。
- PDF/DOC：登记元数据和对象存储 key，深度文本抽取后续由解析 worker 补齐。

该策略避免在 LoongArch + 银河麒麟部署早期引入 native parser、CGO 或外部二进制依赖。

## LoongArch 注意事项

- 当前实现仅使用 Go 标准库和现有纯 Go PostgreSQL 驱动，保持 `CGO_ENABLED=0` 交叉编译。
- Word/PDF/OCR 深度解析属于高风险能力，后续引入前必须验证 linux/loong64 可用性，并保留人工查看原始文件入口。
- 上传文件落盘到 `STORAGE_ROOT/artifacts/...`，生产环境需确保 systemd 用户拥有该目录写权限。
