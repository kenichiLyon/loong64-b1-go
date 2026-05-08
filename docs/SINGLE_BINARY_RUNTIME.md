# 单二进制运行方案

本方案用于把现有系统从“后端二进制 + 前端静态包 + PostgreSQL-only”推进到“单二进制可直接运行，默认 SQLite，可切换 PostgreSQL”的交付形态。

## 1. 目标

- 发布一个可直接运行的主程序二进制，默认内嵌 PC Web 前端。
- 默认无需外部数据库即可启动，首次运行自动落到本地 SQLite。
- 保留 PostgreSQL 作为生产模式数据库。
- 数据库切换不做成日常运行期热切换，而是做成“首次启动 / 管理员设置向导”，写入本地配置后重启生效。

## 2. 现状约束

- 当前前端通过 `vite` 独立构建，后端只提供 API，不托管 `web/dist`。
- 当前数据库层只有 `pgx` + PostgreSQL，一整套迁移、仓储实现和健康检查都绑定 Postgres。
- 现有迁移大量使用 `jsonb`、`timestamptz`、部分索引语法、`FOR UPDATE`、`RETURNING`、`plpgsql trigger`，不能直接在 SQLite 上执行。

结论：

- “内嵌前端”是低风险改造，可以优先完成。
- “默认 SQLite”不是只改一个驱动开关，需要单独做数据库抽象和方言收敛。

## 3. 推荐落地顺序

### 阶段 A：单二进制托管前端

状态：已完成

目标：不改业务逻辑，只让一个二进制同时提供 API 和 Web UI。

改造：

- 在后端增加 `go:embed`，打包 `web/dist`。
- `internal/api` 对 `/api`、`/health` 继续走现有路由，其余路径走静态文件或 SPA fallback `index.html`。
- 开发态保留 `vite dev server` 代理方式；发布态默认使用内嵌前端。
- CI 增加一类用户向 bundle：
  - `loong64-b1-go-full-linux-loong64.tar.gz`
  - `loong64-b1-go-backend-linux-loong64.tar.gz`
  - `loong64-b1-go-frontend.tar.gz`

验收：

- 只启动服务二进制，即可在浏览器直接打开 UI。
- `/api`、`/health`、SPA 路由刷新不冲突。

当前实现说明：

- 默认 `go build ./cmd/server` 仍构建纯后端二进制。
- 在执行 `npm run build --prefix web` 后，使用 `go build -tags webui ./cmd/server` 可构建内嵌前端的完整二进制。
- CI 与 Release 现在对外只发布 3 个 bundle，其中 `full` 是主交付，`backend/frontend` 只用于明确需要分离部署的场景。

### 阶段 B：数据库运行时抽象

状态：已完成基础落地

目标：让服务支持 `sqlite` 与 `postgres` 两种后端。

改造：

- 配置新增：
  - `DB_DRIVER=sqlite|postgres`
  - `SQLITE_PATH=./data/loong64-b1-go.db`
  - `DATABASE_URL=...` 仅在 `postgres` 模式下需要
- `internal/database` 从 `*pgxpool.Pool` 升级为统一运行时抽象，至少暴露：
  - `Driver()`
  - `Ping(ctx)`
  - `Close()`
  - `Query/Exec/Begin` 所需最小接口
- 迁移器拆成方言感知：
  - `migrations/postgres/*.sql`
  - `migrations/sqlite/*.sql`
- 业务仓储拆成：
  - 共享服务接口
  - `postgres` 实现
  - `sqlite` 实现

验收：

- `DB_DRIVER=sqlite` 时，无外部数据库也能完成最小主链路。
- `DB_DRIVER=postgres` 时，保留现有能力。

当前实现说明：

- 配置已支持 `DB_DRIVER=sqlite|postgres`、`SQLITE_PATH` 和 `RUNTIME_CONFIG_PATH`。
- SQLite 模式下服务启动默认执行自动迁移，直接运行二进制即可完成本地初始化。
- `internal/database` 已支持 PostgreSQL / SQLite 双驱动。
- `cmd/migrate` 已按驱动选择根迁移目录或 `migrations/sqlite`。
- `internal/teaching` 已接入 `SQLiteRepository`，并通过集成测试覆盖管理员建课、教师建任务、学生建提交最小链路。
- 已提供管理员运行配置 API 与 PC Web 面板，可把数据库运行参数写入 `runtime.json`。
- 完整的首次启动向导、首个管理员创建引导和无请求头初始化页面仍未完成。

### 阶段 C：首次启动设置向导

状态：已完成首个完整可用版本

目标：通过前端完成数据库模式选择，而不是要求用户先手工写环境变量。

改造：

- 新增本地配置文件，例如 `./config/runtime.json`。
- 如果未初始化，根页面进入 bootstrap wizard：
  - 选择 `SQLite（默认）` 或 `PostgreSQL`
  - 填写 SQLite 路径或 PostgreSQL 连接信息
  - 测试连接
  - 执行初始化迁移
  - 创建首个管理员
- 配置修改后提示“需要重启应用”，不做运行期热切换。

验收：

- 全新目录下直接启动二进制，可以通过浏览器完成初始化。
- 初始化完成后，服务进入正常登录/业务界面。

当前实现说明：

- 已提供管理员运行配置 API 与 PC Web 面板，可把数据库运行参数写入 `runtime.json`。
- 保存配置后会明确提示“需要重启生效”，不做热切换。
- 当数据库中还没有任何用户时，PC Web 会进入 bootstrap 卡片，允许直接创建首个管理员。
- 完整的无认证初始化向导、多步骤数据库切换引导和正式登录体系仍未完成。

## 4. SQLite 选择原则

- 必须避免 CGO，保持 LoongArch 目标一致性。
- 优先选择纯 Go SQLite 驱动。
- 迁移和仓储层不能继续依赖 Postgres 专有 SQL 语法。

候选方向：

- 优先评估 `modernc.org/sqlite` 这类纯 Go 驱动。
- 如果某些约束在 SQLite 上无法等价表达，优先退化为应用层校验，而不是引入 CGO。

## 5. 需要提前收敛的差异

- `jsonb` -> SQLite 中改为 `text/json` + 应用层校验。
- `timestamptz` -> 统一为 RFC3339 UTC 文本或兼容时间列。
- `plpgsql trigger` -> 改为应用层不可变校验，或用 SQLite trigger 重写。
- 部分索引和 `WHERE` 条件索引 -> 分方言保留。
- `FOR UPDATE` 和部分事务语义 -> 在 SQLite 路径单独评估锁粒度。

## 6. 我建议的下一步

先做阶段 A，再做阶段 B，不要把“内嵌前端”和“SQLite 支持”绑定到同一个大提交里。

原因：

- 阶段 A 改造面小，能立即提升交付体验。
- 阶段 B 会同时动配置、数据库抽象、迁移、仓储和测试，风险明显高。
- 先完成阶段 A 后，应用已经更接近“单文件可运行”，再逐步把数据库从外置必选改成默认内置。
