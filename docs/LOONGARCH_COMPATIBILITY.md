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

阶段 3 成果上传与解析未新增第三方运行时依赖；文件嗅探、SHA-256、ZIP 安全检查、图片元数据和文本摘要均使用 Go 标准库。阶段 4 规则核查与 LLM 初评继续使用 Go 标准库 HTTP/JSON/regexp/crypto 能力和已有 pgx 依赖。阶段 5 教师复核与发布仅新增 SQL/JSON/权限校验逻辑，不引入 CGO、tokenizer、OCR、浏览器驱动或本地模型运行时。阶段 6 报表使用 Go 标准库生成 HTML/CSV，PDF 在未完成 LoongArch 渲染器与中文字体验证前只记录降级状态。阶段 7 增加部署验证脚本、Nginx 静态托管示例和环境采样流程，仍不引入新的运行时依赖。深度 Word/PDF/OCR 解析和正式 PDF 生成继续列为 LoongArch 高风险能力，后续必须逐项验证。

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
| 成果上传落盘 | 本地 ObjectStore | 低 | 限制大小/数量，保留 SHA-256 和审计 |
| ZIP/代码包检查 | Go 标准库 `archive/zip` | 低 | 拒绝路径穿越、符号链接和超大解压 |
| Word/PDF 解析 | 纯 Go 优先，可替换外部工具 | 高 | 解析失败时保留原文查看和人工复核 |
| 截图 OCR | 可选能力 | 高 | OCR 不可用时只保存图片和人工查看 |
| Excel 导出 | Stage 6 MVP 使用 UTF-8 BOM CSV；后续纯 Go XLSX 库 | 中 | 导出 CSV 或简化 XLSX |
| PDF 导出 | HTML 规范视图 + PDF 模板；MVP 先记录待配置 | 高 | 图表降级为表格，保留 HTML 归档 |
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

推荐直接执行：

```bash
sh deploy/kylin/scripts/collect-env.sh /tmp/loong64-b1-go-stage7.txt
```


## 前端静态资源

Stage 5.5 前端在开发或 CI 环境中使用 Node/Vite 构建，LoongArch + 银河麒麟目标机默认只托管 `web/dist` 静态产物，不要求目标机安装 Node.js。若必须在目标机重新构建，需要单独验证 Node.js 与 npm 依赖在 LoongArch 上可用。Stage 7 默认通过 `deploy/kylin/nginx/loong64-b1-go.conf.example` 提供静态站点与 `/api`、`/health` 反向代理示例。
