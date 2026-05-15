# 自动代码审核流水线

本项目在 Auto Build 和 CD Publish Artifacts 之外，增加 `Code Quality Review` 工作流，用于 PR 和分支推送时的自动代码审核。

## 触发条件

文件：`.github/workflows/code-quality.yml`

- push 到 `main`、`feature/**`、`fix/**`
- pull request 到 `main`
- 手动 `workflow_dispatch`

## Go Linter

`go-lint` job 使用 `golangci/golangci-lint-action`，并读取 `.golangci.yml`：

- 使用 GitHub Actions 当前稳定 Go 工具链。
- 使用 `golangci-lint` v2.11.4。
- 默认开启 `standard` linter 集合。
- 超时时间为 5 分钟。
- lint 失败会阻断 PR 合并。

## SourceryAI

`sourceryai` job 仅在 PR 中运行，用于补充 AI 代码审核建议。

仓库需要配置 GitHub Actions Secret：

```text
SOURCERY_TOKEN
```

未配置 `SOURCERY_TOKEN` 时，工作流会跳过 SourceryAI，不阻断其他 CI 检查。配置后，SourceryAI 会基于 PR base commit 进行差异审核。

`check: false` 表示 SourceryAI 建议不作为硬性失败条件；Go linter、测试和构建仍是阻断式质量门槛。

## 与发布流水线关系

- `Code Quality Review` 负责自动代码审核和 lint。
- `Auto Build` 负责格式检查、测试和构建产物。
- `CD Publish Artifacts` 只发布 Auto Build 产物，不重新编译。

## 本地验证

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 config verify
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run
go test ./...
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/server
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build ./cmd/upgrade
```
