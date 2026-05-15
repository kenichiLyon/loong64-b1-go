# 数据库迁移

PostgreSQL 迁移脚本放在本目录，并由 `cmd/upgrade` 按文件名前缀顺序执行。

## 运行

```bash
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable go run ./cmd/upgrade up
```

Windows PowerShell：

```powershell
$env:DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable'; go run ./cmd/upgrade up
```

## 命名

```text
000001_foundation.sql
000002_teaching_domain.sql
000003_submission_artifacts.sql
000004_evaluation_initial.sql
000005_teacher_review_publish.sql
000006_report_exports.sql
000007_report_course_summary.sql
000008_assistant_context.sql
```

## 当前迁移说明

- `000001_foundation.sql`：应用元数据、异步任务和审计日志。
- `000002_teaching_domain.sql`：用户、角色、班级、课程、选课、评价模板和实训任务。
- `000003_submission_artifacts.sql`：学生提交、附件元数据、解析结果和上传解析任务队列。
- `000004_evaluation_initial.sql`：规则核查、LLM 初评、指标建议分和调用日志。
- `000005_teacher_review_publish.sql`：教师复核、逐指标最终分、发布与不可变约束。
- `000006_report_exports.sql`：报表导出任务记录、筛选条件、对象存储 key、hash 和状态。
- `000007_report_course_summary.sql`：扩展报表导出约束，支持 `course_summary` 报表类型和 `course` 作用域。
- `000008_assistant_context.sql`：部署助手会话、消息、上下文快照、受控工具调用和 LLM 调用日志。

迁移文件一旦提交，不得修改已应用文件内容；需要变更时新增下一个版本。
