# 数据库迁移

PostgreSQL 迁移脚本放在本目录，并由 `cmd/migrate` 按文件名前缀顺序执行。

## 运行

```bash
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable go run ./cmd/migrate up
```

Windows PowerShell：

```powershell
$env:DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/loong64_b1?sslmode=disable'; go run ./cmd/migrate up
```

## 命名

```text
000001_foundation.sql
000002_teaching_domain.sql
000003_submission_artifacts.sql
```

迁移文件一旦提交，不得修改已应用文件内容；需要变更时新增下一个版本。
