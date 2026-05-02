# LoongArch 与银河麒麟兼容性记录

## 固定目标环境

- CPU 架构：LoongArch / `loongarch64`
- Go 目标：`GOOS=linux GOARCH=loong64 CGO_ENABLED=0`
- 操作系统：银河麒麟高级服务器版
- 部署方式：systemd 非容器部署为主，Podman/Docker 为辅
- 数据库：PostgreSQL

## 当前策略

- 后端默认纯 Go，避免 CGO 和预编译 native binary。
- 交叉编译作为每次后端变更的最低门槛。
- 最终验收必须在 LoongArch 真机或等价环境运行 smoke test。
- 文档解析、OCR、PDF 和本地模型服务是高风险项，必须逐项验证并保留降级入口。

## 当前依赖评估

| 依赖 | 用途 | CGO | LoongArch 风险 | 处理 |
| --- | --- | --- | --- | --- |
| Go 标准库 | HTTP 服务、配置、日志 | 否 | 低 | 持续交叉编译 |
| `github.com/jackc/pgx/v5` | PostgreSQL 连接池 | 否 | 低-中 | 固定版本，使用 `CGO_ENABLED=0` 验证 |

阶段 2 教学域未新增第三方运行时依赖；数据库、API、权重校验和权限校验继续基于 Go 标准库与既有 `pgx/v5`。

## 必跑检查

```bash
go test ./...
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/migrate
```

Windows PowerShell：

```powershell
$env:GOOS='linux'; $env:GOARCH='loong64'; $env:CGO_ENABLED='0'; go build ./cmd/server; go build ./cmd/migrate; Remove-Item Env:GOOS,Env:GOARCH,Env:CGO_ENABLED
```

## 能力评估表

| 能力 | 默认方案 | LoongArch 风险 | 降级策略 |
| --- | --- | --- | --- |
| Go API 服务 | 纯 Go 标准库优先 | 低 | 禁用 CGO，目标机 smoke test |
| PostgreSQL | 系统包或官方包 | 中 | 记录安装来源，提供备份恢复脚本 |
| Word/PDF 解析 | 纯 Go 优先，可替换外部工具 | 高 | 解析失败时保留原文查看和人工复核 |
| 截图 OCR | 可选能力 | 高 | OCR 不可用时只保存图片和人工查看 |
| Excel 导出 | 纯 Go XLSX 库 | 中 | 导出 CSV 或简化 XLSX |
| PDF 导出 | HTML 规范视图 + PDF 模板 | 高 | 图表降级为表格，保留 HTML 归档 |
| 本地模型服务 | OpenAI-compatible HTTP | 高 | 使用学校内网或云端模型网关 |
| 容器部署 | Podman/Docker 可选 | 中 | systemd 非容器部署为主 |

## 目标机验证记录模板

追加记录时使用以下格式：

```text
日期：YYYY-MM-DD
机器：LoongArch / loongarch64
系统：银河麒麟高级服务器版 <版本>
内核：<uname -a>
Go：<go version>
PostgreSQL：<psql --version>
验证命令：<commands>
结果：通过 / 失败
问题与处理：<notes>
```
