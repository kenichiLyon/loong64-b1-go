# 阶段 2 教学域接口与数据模型

本阶段补齐用户、课程、班级、选课、评价模板和实训任务的后端基础闭环，为后续成果上传、核查、评分和报表提供稳定数据边界。

## 1. 系统升级迁移

新增迁移：`migrations/000002_teaching_domain.sql`

核心表：

- `users`、`user_roles`：账号与多角色。
- `classes`、`courses`、`course_classes`：班级、课程和课程班级关系。
- `course_teachers`、`enrollments`：教师授权和学生选课。
- `rubric_templates`、`rubric_template_versions`、`rubric_metrics`：评价模板、版本和指标。
- `experiments`：实训任务，绑定具体已发布模板版本。

## 2. 权重与版本规则

评价指标权重使用整数基点：

```text
100% = 10000
```

支持两种模式：

- `strict_100`：发布模板版本时，指标权重总和必须等于 `10000`。
- `normalized`：发布模板版本时，指标权重总和必须大于 `0`，评分时按比例归一化。

版本规则：

- 教师只能修改自己创建的模板，管理员不受此限制。
- 模板版本创建后为 `draft`。
- 发布版本前会重新聚合校验指标权重。
- `published` 的模板版本和指标由数据库触发器禁止更新或删除。
- 实训任务绑定 `rubric_template_versions.id`，不绑定模板主表，保证历史评价可追溯。

## 3. 临时开发认证

阶段 2 尚未接入正式登录。API 使用以下开发头传入操作者：

```text
X-Actor-ID: teacher-1
X-Actor-Roles: teacher
```

如确需本机快速冒烟，可显式设置：

```text
DEV_AUTH_BYPASS=true
```

此开关只在非 `production` 环境且请求来自 loopback 地址时生效。启用后，未传入请求头时会使用内置开发操作者并写入警告日志：

```text
X-Actor-ID: dev-admin
X-Actor-Roles: admin,teacher,student
```

生产环境必须显式传入操作者头；`DEV_AUTH_BYPASS` 必须保持 `false`。后续阶段将替换为正式认证和会话。

## 4. API 边界

管理员接口：

- `GET /api/v1/admin/users`
- `POST /api/v1/admin/users`
- `PUT /api/v1/admin/users/{userID}/roles`
- `POST /api/v1/admin/classes`
- `POST /api/v1/admin/courses`
- `PUT /api/v1/admin/courses/{courseID}/classes`
- `PUT /api/v1/admin/courses/{courseID}/teachers`
- `PUT /api/v1/admin/courses/{courseID}/enrollments`

教师接口：

- `POST /api/v1/teacher/rubric-templates`
- `POST /api/v1/teacher/rubric-templates/{templateID}/versions`
- `POST /api/v1/teacher/rubric-template-versions/{versionID}/publish`
- `POST /api/v1/teacher/courses/{courseID}/experiments`
- `POST /api/v1/teacher/experiments/{experimentID}/publish`
- `GET /api/v1/teacher/experiments/{experimentID}/submissions`
- `GET /api/v1/teacher/submissions/{submissionID}`

学生提交接口：

- `POST /api/v1/student/experiments/{experimentID}/submissions`
- `POST /api/v1/student/submissions/{submissionID}/artifacts`
- `POST /api/v1/student/submissions/{submissionID}/artifact-links`
- `GET /api/v1/student/submissions/{submissionID}`

成果上传与解析详见 `docs/SUBMISSION_UPLOAD.md`。

## 5. 最小冒烟流程

1. 管理员创建教师、学生、班级和课程。
2. 管理员将班级关联到课程。
3. 管理员给课程分配教师。
4. 管理员把学生加入课程。
5. 教师创建评价模板。
6. 教师创建模板版本和指标。
7. 教师发布模板版本。
8. 教师创建实训任务并绑定已发布模板版本。
9. 教师发布实训任务。

## 6. LoongArch 影响

本阶段只使用 Go 标准库和既有 `pgx/v5`，不新增 CGO 或原生二进制依赖。仍需保持：

```bash
go test ./...
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/upgrade
```
