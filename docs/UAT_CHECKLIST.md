# UAT 执行手册

这份文档用于试点 MVP 的人工验收。目标不是“打勾”，而是保证每一步都能留下可复现记录。

可选自动化 smoke：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/uat/run-local-uat.ps1
```

这条脚本优先覆盖本地 SQLite + 单服务二进制场景，用于先验证核心 API 闭环；目标机实测和视觉验收仍按下文逐项执行。

## 0. 前置条件

- 已完成 `go test ./...`
- 已完成前端构建：`cd web && npm run build`
- 已完成后端交叉编译：`GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server` 与 `go build ./cmd/migrate`
- 目标环境可访问 `/health/live` 与 `/health/ready`
- 已准备一个干净的测试库或临时数据目录，避免与旧数据混用

## 1. 执行方式

建议按下面顺序执行，并在每一步后记录结果：

1. 初始化
2. 管理搭建
3. 教学配置
4. 提交与评价
5. 报表与导出
6. 安全与会话
7. 部署检查

## 2. 初始化

- [ ] 打开首页，未初始化时显示 bootstrap 面板
- [ ] 创建首个管理员
- [ ] 创建后自动登录
- [ ] `GET /api/v1/me` 返回 admin

执行记录：

```text
时间：
环境：
操作：
结果：
证据：
```

## 3. 管理搭建

- [ ] 创建 teacher 用户
- [ ] 创建 student 用户
- [ ] 创建班级
- [ ] 创建课程
- [ ] 课程关联班级
- [ ] 课程分配教师
- [ ] 学生登记选课

建议核对接口：

- `POST /api/v1/admin/users`
- `POST /api/v1/admin/classes`
- `POST /api/v1/admin/courses`
- `PUT /api/v1/admin/courses/{courseID}/classes`
- `PUT /api/v1/admin/courses/{courseID}/teachers`
- `PUT /api/v1/admin/courses/{courseID}/enrollments`

执行记录：

```text
时间：
创建的 user/class/course ID：
关系结果：
异常：
证据：
```

## 4. 教学配置

- [ ] 教师创建评价模板
- [ ] 教师创建模板版本
- [ ] 教师发布模板版本
- [ ] 教师创建实验
- [ ] 教师发布实验

建议核对接口：

- `POST /api/v1/teacher/rubric-templates`
- `POST /api/v1/teacher/rubric-templates/{templateID}/versions`
- `POST /api/v1/teacher/rubric-template-versions/{versionID}/publish`
- `POST /api/v1/teacher/courses/{courseID}/experiments`
- `POST /api/v1/teacher/experiments/{experimentID}/publish`

执行记录：

```text
时间：
template/version/experiment ID：
发布状态：
证据：
```

## 5. 提交与评价

- [ ] 学生创建提交
- [ ] 上传 `.txt` 或 `.md` 报告成功并生成文本摘要
- [ ] 上传 `.docx` 成功并生成正文摘要
- [ ] 上传 `.pdf` 成功并生成页数或文本摘要
- [ ] 上传截图成功并记录图片尺寸
- [ ] 上传 `.zip` 代码包成功并记录清单摘要
- [ ] 教师触发规则初评成功
- [ ] 教师保存复核草稿成功
- [ ] 教师发布最终评价成功
- [ ] 学生查看已发布评价成功

建议核对接口：

- `POST /api/v1/student/experiments/{experimentID}/submissions`
- `POST /api/v1/student/submissions/{submissionID}/artifacts`
- `POST /api/v1/student/submissions/{submissionID}/artifact-links`
- `POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial`
- `PUT /api/v1/teacher/submissions/{submissionID}/review`
- `POST /api/v1/teacher/submissions/{submissionID}/review/publish`
- `GET /api/v1/student/submissions/{submissionID}/review`

执行记录：

```text
时间：
submission ID：
artifact ID：
初评结果：
复核结果：
发布结果：
证据：
```

## 6. 报表与导出

- [ ] 查看个人评价报告
- [ ] 查看实验统计
- [ ] 查看课程统计
- [ ] 导出 HTML 成功
- [ ] 导出 CSV 成功
- [ ] 导出 XLSX 成功
- [ ] 导出 PDF 成功

建议核对接口：

- `GET /api/v1/teacher/submissions/{submissionID}/report`
- `GET /api/v1/student/submissions/{submissionID}/report`
- `GET /api/v1/teacher/experiments/{experimentID}/reports/summary`
- `GET /api/v1/teacher/courses/{courseID}/reports/summary`
- `POST /api/v1/teacher/submissions/{submissionID}/report-exports`
- `POST /api/v1/teacher/experiments/{experimentID}/report-exports`
- `POST /api/v1/teacher/courses/{courseID}/report-exports`

执行记录：

```text
时间：
导出类型：
导出 ID：
存储 key：
下载结果：
证据：
```

## 7. 安全与会话

- [ ] 登录成功后下发 session cookie
- [ ] 不带 `X-CSRF-Token` 的写请求返回 `403`
- [ ] 跨站 `Origin` 的写请求返回 `403`
- [ ] 自助改密后旧 session 失效
- [ ] 管理员重置密码后该用户旧 session 失效

建议核对接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `PUT /api/v1/auth/password`
- `PUT /api/v1/admin/users/{userID}/password`

执行记录：

```text
时间：
会话 cookie：
CSRF cookie：
拒绝原因：
旧 session 失效情况：
证据：
```

## 8. 部署检查

- [ ] `go test ./...` 通过
- [ ] `npm run build --prefix web` 通过
- [ ] `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server` 通过
- [ ] `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate` 通过
- [ ] 目标机 `/health/live` 与 `/health/ready` 返回正常

建议在目标机记录：

```text
日期：
机器名：
uname -m：
银河麒麟版本：
内核版本：
Go 版本：
PostgreSQL 版本：
字体：
服务状态：
```

## 9. 结论模板

```text
结论：通过 / 有条件通过 / 未通过
阻塞项：
修复建议：
负责人：
复验时间：
```
