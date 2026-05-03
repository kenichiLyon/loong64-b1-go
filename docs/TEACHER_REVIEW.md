# 教师复核与成绩发布

阶段 5 将规则核查和 LLM 初评转换为教师可确认的最终评价结果。AI 与规则分只作为建议源，教师保存草稿、逐指标改分、填写评语并显式发布后，学生才可查看。

## API

- `PUT /api/v1/teacher/submissions/{submissionID}/review`
  - 保存或更新教师复核草稿。
  - 请求必须包含 Rubric 下所有指标的最终分。
  - 可引用 `evaluation_result_id` 和 `source_metric_score_id`，用于记录“从 AI/规则建议调整而来”。
- `POST /api/v1/teacher/submissions/{submissionID}/review/publish`
  - 发布最终评价，请求体必须包含 `{ "confirm": true }`。
  - 发布后草稿和逐指标最终分不可修改。
- `GET /api/v1/teacher/submissions/{submissionID}/review`
  - 教师查看草稿或已发布评价。
- `GET /api/v1/student/submissions/{submissionID}/review`
  - 学生只可查看自己的已发布评价；未发布时不暴露教师草稿。

## 数据表

- `teacher_reviews`：每个提交一条教师复核主记录，保存状态、总分、评语、发布人和发布时间。
- `teacher_metric_scores`：每个指标的最终分、来源、调整理由和教师说明。

数据库触发器会阻止已发布 `teacher_reviews` 和其下 `teacher_metric_scores` 被更新或删除，保证后台初评重跑不会覆盖学生可见成绩。

## 评分规则

- `total_score_bps` 使用整数基点，`10000 = 100%`。
- 每个指标必须提交最终分，且 `0 <= final_score <= max_score`。
- 总分按指标权重加权：`sum(final_score / max_score * weight_bps)`。
- `source` 可为 `manual`、`rule`、`llm`，仅表示教师参考来源，不代表最终成绩由模型自动决定。

## 权限与发布边界

- 管理员和被授权教师可保存、查看、发布复核结果。
- 学生只能查看自己的已发布结果。
- 已发布结果不可通过复核接口再次保存；如需更正，需要后续阶段设计正式更正/申诉流程并单独审计。
- 所有保存和发布动作写入 `audit_logs`，审计 detail 只保存提交、复核和建议来源 ID，不保存原始学生文件或完整 Prompt。

## 验证

```bash
go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate
```