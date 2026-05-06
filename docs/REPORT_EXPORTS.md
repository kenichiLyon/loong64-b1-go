# 报表与导出

Stage 6 当前提供个人评价报告、实验统计摘要、课程跨实验统计、HTML/CSV 导出和 PDF 降级记录。实现原则是 LoongArch + 银河麒麟优先：默认不引入浏览器、LibreOffice headless、wkhtmltopdf 或 CGO/native 依赖。

## 能力范围

- 个人报告：聚合提交、实验、附件解析摘要、教师发布/草稿复核、最新智能核查结果。
- 实验统计：提交数、已提交数、已发布评价数、平均/最高/最低分、分数段、指标均值、附件解析状态和常见问题统计。
- 课程统计：聚合课程下多个实验的提交、发布评分、分数分布、指标均值、常见问题和实验子摘要。
- HTML 导出：作为规范归档源，写入对象存储 `reports/` 目录。
- CSV 导出：带 UTF-8 BOM，便于 WPS、LibreOffice 和 Microsoft Excel 直接打开。
- PDF 导出：当前创建导出记录并标记失败/待配置，错误信息说明需要先验证 LoongArch 渲染器和中文字体链路。

## API

```text
GET  /api/v1/teacher/submissions/{submissionID}/report
GET  /api/v1/student/submissions/{submissionID}/report
GET  /api/v1/teacher/experiments/{experimentID}/reports/summary
GET  /api/v1/teacher/courses/{courseID}/reports/summary
POST /api/v1/teacher/submissions/{submissionID}/report-exports
POST /api/v1/teacher/experiments/{experimentID}/report-exports
POST /api/v1/teacher/courses/{courseID}/report-exports
GET  /api/v1/teacher/report-exports/{exportID}
GET  /api/v1/teacher/report-exports/{exportID}/download
```

导出请求示例：

```json
{
  "format": "csv"
}
```

`format` 支持：

- `html`：生成可归档的 HTML 报告。
- `csv`：生成 Excel/WPS/LibreOffice 兼容 CSV。
- `pdf`：记录为失败/待配置，不生成二进制 PDF。

## 权限

- 教师/管理员：可查看有课程授权的提交报告、实验统计和课程统计，可创建自己的导出任务。
- 学生：只能查看自己提交且已发布评价的个人报告。
- 下载：教师只能下载自己创建的导出；管理员可查看全部导出记录。

## 存储与审计

- `report_exports` 保存报表类型、范围、格式、状态、筛选条件、操作者、文件 key、SHA-256 和字节数。
- 文件保存到 `STORAGE_ROOT/reports/{scope_type}/{scope_id}/{export_id}.{html|csv}`，其中 `scope_type` 支持 `submission`、`experiment`、`course`。
- 创建导出写入 `audit_logs`，动作为 `report_export.create`。
- 当前实现同步生成 HTML/CSV，同时写入 `jobs(job_type='report_export')` 保留后续异步 worker 接口。

## LoongArch PDF 策略

PDF 中文与图表在 LoongArch + 银河麒麟上存在字体、渲染器、外部二进制和并发稳定性风险。后续启用 PDF 前必须完成：

- 选定纯 Go 或目标机已验证渲染方案。
- 明确中文字体路径、许可证和缺失降级策略。
- 在目标机验证生成、打开、中文不乱码。
- 图表渲染失败时自动降级为表格摘要。

在上述条件满足前，HTML 是归档源，CSV 是 Excel 兼容交付物。
