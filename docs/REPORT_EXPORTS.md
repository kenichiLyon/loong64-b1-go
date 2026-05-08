# 报表与导出

Stage 6 当前提供个人评价报告、实验统计摘要、课程跨实验统计，以及 HTML/CSV/XLSX/PDF 导出。实现原则仍然是 LoongArch + 银河麒麟优先：优先纯 Go 依赖，不引入浏览器、LibreOffice headless、wkhtmltopdf 或 CGO/native 依赖。

## 能力范围

- 个人报告：聚合提交、实验、附件解析摘要、教师发布/草稿复核、最新智能核查结果。
- 实验统计：提交数、已提交数、已发布评价数、平均/最高/最低分、分数段、指标均值、附件解析状态和常见问题统计。
- 课程统计：聚合课程下多个实验的提交、发布评分、分数分布、指标均值、常见问题和实验子摘要。
- HTML 导出：作为规范归档源，写入对象存储 `reports/` 目录。
- CSV 导出：带 UTF-8 BOM，便于 WPS、LibreOffice 和 Microsoft Excel 直接打开。
- XLSX 导出：生成原生 Excel 工作簿，适合教师汇总与二次分析。
- PDF 导出：使用内嵌中文字体生成归档版报告，适合打印与提交。

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
  "format": "xlsx"
}
```

`format` 支持：

- `html`：生成可归档的 HTML 报告。
- `csv`：生成 Excel/WPS/LibreOffice 兼容 CSV。
- `xlsx`：生成原生 Excel 工作簿。
- `pdf`：生成带中文字体的 PDF 文档。

## 权限

- 教师/管理员：可查看有课程授权的提交报告、实验统计和课程统计，可创建自己的导出任务。
- 学生：只能查看自己提交且已发布评价的个人报告。
- 下载：教师只能下载自己创建的导出；管理员可查看全部导出记录。

## 存储与审计

- `report_exports` 保存报表类型、范围、格式、状态、筛选条件、操作者、文件 key、SHA-256 和字节数。
- 文件保存到 `STORAGE_ROOT/reports/{scope_type}/{scope_id}/{export_id}.{html|csv|xlsx|pdf}`，其中 `scope_type` 支持 `submission`、`experiment`、`course`。
- 创建导出写入 `audit_logs`，动作为 `report_export.create`。
- 当前实现同步生成 HTML/CSV/XLSX/PDF，同时写入 `jobs(job_type='report_export')` 保留后续异步 worker 接口。

## LoongArch PDF 策略

当前 PDF 采用纯 Go `gofpdf` 与内嵌中文字体，不依赖目标机浏览器或 office 套件。后续仍需在目标机继续关注：

- 大文件报告生成耗时与内存占用。
- 字体体积对完整二进制大小的影响。
- 更复杂图表、分页和模板样式的稳定性。
