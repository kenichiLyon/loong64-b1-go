# UAT 检查单

## 初始化

- [ ] 访问首页，系统未初始化时显示 bootstrap 面板
- [ ] 创建首个管理员成功
- [ ] 自动登录成功，`/api/v1/me` 返回 admin

## 管理搭建

- [ ] 创建 teacher 用户
- [ ] 创建 student 用户
- [ ] 创建班级
- [ ] 创建课程
- [ ] 课程关联班级
- [ ] 课程分配教师
- [ ] 学生登记选课

## 教学配置

- [ ] 教师创建评价模板
- [ ] 教师创建模板版本
- [ ] 教师发布模板版本
- [ ] 教师创建实验
- [ ] 教师发布实验

## 提交与评价

- [ ] 学生创建提交
- [ ] 上传 `.txt/.md` 报告成功并生成文本摘要
- [ ] 上传 `.docx` 成功并生成正文摘要
- [ ] 上传 `.pdf` 成功并生成页数/文本摘要
- [ ] 上传截图成功并记录图片尺寸
- [ ] 上传 `.zip` 代码包成功并记录清单摘要
- [ ] 教师触发规则初评成功
- [ ] 教师保存复核草稿成功
- [ ] 教师发布最终评价成功
- [ ] 学生查看已发布评价成功

## 报表与导出

- [ ] 查看个人评价报告
- [ ] 查看实验统计
- [ ] 查看课程统计
- [ ] 导出 HTML 成功
- [ ] 导出 CSV 成功
- [ ] 导出 XLSX 成功
- [ ] 导出 PDF 成功

## 安全与会话

- [ ] 登录成功后下发 session 和 csrf cookie
- [ ] 不带 `X-CSRF-Token` 的写请求返回 `403`
- [ ] 跨站 `Origin` 的写请求返回 `403`
- [ ] 自助改密后旧 session 失效
- [ ] 管理员重置用户密码后该用户旧 session 失效

## 部署

- [ ] `go test ./...` 通过
- [ ] `npm run build --prefix web` 通过
- [ ] `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server` 通过
- [ ] `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate` 通过
- [ ] 目标机 `/health/live` 与 `/health/ready` 返回正常
