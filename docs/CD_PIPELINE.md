# CD 发布流水线

本项目使用 GitHub Actions 管理自动代码审核、自动构建和发布。流水线必须只发布由 Auto Build 产生的构建产物，不在 CD 阶段重新编译，确保发布内容可追溯。

## Code Quality Review

文件：`.github/workflows/code-quality.yml`

触发：

- push 到 `main`、`feature/**` 或 `fix/**`
- pull request 到 `main`
- 手动 `workflow_dispatch`

步骤：

1. 使用 `golangci-lint` 执行 Go 静态检查，读取 `.golangci.yml`。
2. 使用 Node 24 和 `web/package-lock.json` 安装前端依赖，运行 `npm run build --prefix web`，同时完成 Vue/TypeScript 类型检查和 Vite 构建。
3. PR 场景下，如果仓库配置了 `SOURCERY_TOKEN`，运行 SourceryAI 差异代码审核。
4. 未配置 `SOURCERY_TOKEN` 时只跳过 SourceryAI，不影响 Go linter 和前端构建。

详见 `docs/CODE_REVIEW_CI.md`。

## Auto Build

文件：`.github/workflows/auto-build.yml`

触发：

- push 到 `main` 或 `feature/**`
- pull request 到 `main`
- 手动 `workflow_dispatch`

步骤：

1. 检出代码。
2. 使用 GitHub Actions 当前稳定 Go 工具链。
3. 检查 `gofmt`。
4. 运行 `go test ./...`。
5. 使用 Node 24 执行 `npm ci --prefix web` 与 `npm run build --prefix web`。
6. 构建 `linux/loong64` 纯后端服务端二进制、内嵌前端完整二进制和迁移二进制。
7. 调用 `scripts/release/package-release.sh` 组装 3 个面向用户的发布 bundle：
   - `loong64-b1-go-full-linux-loong64.tar.gz`
   - `loong64-b1-go-backend-linux-loong64.tar.gz`
   - `loong64-b1-go-frontend.tar.gz`
8. 在 artifact 内部生成 `SHA256SUMS`，供 CD release notes 引用，但不作为独立 Release 资产发布。
9. 上传名为 `auto-build-<commit sha>` 的 artifact。

## CD Publish Artifacts

文件：`.github/workflows/cd-release.yml`

触发：

- `main` 分支 Auto Build 成功后自动触发。
- 手动 `workflow_dispatch`，输入需要发布的 Auto Build `run_id`。

步骤：

1. 解析 Auto Build run id 和 head commit。
2. 下载 Auto Build 上传的 artifact。
3. 读取 `SHA256SUMS` 并生成 release notes。
4. 创建 tag：`auto-build-<short sha>`。
5. 创建 GitHub Release，并且只上传 3 个面向用户的 bundle，不上传 `AGENT.md`、`PLAN.md`、原始二进制散件或独立 `SHA256SUMS`。

## 权限

- Code Quality Review：`contents: read`、`pull-requests: read`。
- Auto Build：`contents: read`。
- CD Publish Artifacts：`contents: write`、`actions: read`。

## 约束

- CD 不重新编译，只发布 Auto Build 的产物。
- Release 页面只展示 3 个用户向交付物：`full`、`backend`、`frontend`。
- SourceryAI 需要仓库级 Secret `SOURCERY_TOKEN`；不得把 token 写入仓库。
- Release tag 与 Auto Build commit 绑定。
- 如果同名 Release 已存在，流水线跳过发布，避免覆盖历史产物。
- 发布物不得包含 `.env`、密钥、真实学生数据或本地 storage 目录。
