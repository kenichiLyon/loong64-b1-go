# 本地 UAT 执行记录

日期：

- `2026-05-14`

执行环境：

- 主机：`Windows amd64 开发机`
- 运行方式：`本地 SQLite + Go 主服务 + Python 推理微服务协作链路`
- 说明：本记录用于证明本地主线闭环已跑通；它不能替代 `LoongArch + 银河麒麟` 目标机验收记录。

## 1. 执行命令

本地 UAT smoke：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\uat\run-local-uat.ps1 -Port 18087
```

代码验证：

```bash
go test ./...
npm run build --prefix web
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate
```

## 2. UAT 结果摘要

本次本地 UAT 输出：

```json
{
  "bootstrap": {
    "message": "bootstrap completed and admin session created"
  },
  "teacher_id": "usr_82593fa5416846ddd02b7a03",
  "student_id": "usr_f40753eafc44ca2bdbf44127",
  "class_id": "cls_1bc56ade7b54ddf15880c8b1",
  "course_id": "crs_0c87334d6770373894d6be25",
  "template_id": "rbt_f1b98b2fbc545c0515cc0c79",
  "version_id": "rbv_adbeab4a46718347c95273b7",
  "experiment_id": "exp_bb0570a99cb7f03e02c7beb7",
  "submission_id": "sub_270623343eea08479fecea77",
  "evaluation_id": "evr_d60cda4cc40412c8058426ce",
  "review_status": "draft",
  "published_status": "published",
  "student_review_status": "published",
  "report_export_id": "rpx_0a3451c78e5a6f6dfd605f86",
  "experiment_export_id": "rpx_c13ab59f45685f758bcf085a",
  "course_export_id": "rpx_be2bbd0def7e4eb87438a042",
  "student_experiments": 1
}
```

## 3. 本次已验证内容

- bootstrap 创建首个管理员成功
- 管理员、教师、学生、班级、课程、模板、版本、实验创建成功
- 学生提交创建成功
- 多种成果上传链路打通：
  - `report.md`
  - `report.docx`
  - `report.pdf`
  - `shot.png`
  - `code.zip`
- 教师初评成功
- 教师复核草稿保存成功
- 教师最终发布成功
- 学生读取已发布评价成功
- 导出成功：
  - 个人报告 `HTML`
  - 实验统计 `CSV`
  - 课程统计 `PDF`

## 4. 代码验证结果

- `go test ./...`：通过
- `npm run build --prefix web`：通过
- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server`：通过
- `GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate`：通过

## 5. 本地生成物

本次本地 UAT 生成目录：

- `tmp/uat-29e20b65adf642d681ae73cb77f45bd8`

其中包含：

- SQLite 数据文件
- 样例成果文件
- 对象存储产物
- 导出文件
- 服务日志

这些产物用于本地验证，不作为正式交付物。

## 6. 当前结论

结论：

- `本地主线闭环已验证通过`

仍未完成、因此不能单靠本记录宣布“正式可交付”的部分：

- `LoongArch + 银河麒麟目标机部署验收`
- 目标机 systemd 双服务留档
- 目标环境健康检查与运行记录
- 目标机上实际 UAT 留痕

所以当前状态更准确的表述是：

- `本地可交付候选已形成`
- `正式交付仍需目标机验收闭环`
