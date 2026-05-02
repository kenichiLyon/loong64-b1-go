# CD 发布流水线

本项目使用 GitHub Actions 管理自动构建和发布。流水线必须只发布由 Auto Build 产生的构建产物，不在 CD 阶段重新编译，确保发布内容可追溯。

## Auto Build

文件：`.github/workflows/auto-build.yml`

触发：

- push 到 `main` 或 `feature/**`
- pull request 到 `main`
- 手动 `workflow_dispatch`

步骤：

1. 检出代码。
2. 使用 `go.mod` 配置 Go 工具链。
3. 检查 `gofmt`。
4. 运行 `go test ./...`。
5. 构建 `linux/amd64` 和 `linux/loong64` 服务端二进制。
6. 生成 `SHA256SUMS`。
7. 上传名为 `auto-build-<commit sha>` 的 artifact。

## CD Publish Artifacts

文件：`.github/workflows/cd-release.yml`

触发：

- `main` 分支 Auto Build 成功后自动触发。
- 手动 `workflow_dispatch`，输入需要发布的 Auto Build `run_id`。

步骤：

1. 解析 Auto Build run id 和 head commit。
2. 下载 Auto Build 上传的 artifact。
3. 生成 release notes。
4. 创建 tag：`auto-build-<short sha>`。
5. 创建 GitHub Release 并上传构建产物。

## 权限

- Auto Build：`contents: read`。
- CD Publish Artifacts：`contents: write`、`actions: read`。

## 约束

- CD 不重新编译，只发布 Auto Build 的产物。
- Release tag 与 Auto Build commit 绑定。
- 如果同名 Release 已存在，流水线跳过发布，避免覆盖历史产物。
- 发布物不得包含 `.env`、密钥、真实学生数据或本地 storage 目录。
