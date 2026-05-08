# 部署助手

部署助手是 `loong64-b1-go` 中面向初始化、数据库切换与运行配置排障的持久会话能力。它不依赖模型会话状态；所有上下文由服务端重新构建并持久化。

## 能力范围

- `bootstrap` 作用域
  - 读取当前 bootstrap 状态
  - 读取当前运行配置摘要
  - 在用户确认后创建首个管理员
- `deployment_admin` 作用域
  - 读取当前运行配置与已保存 `runtime.json`
  - 测试 SQLite 路径
  - 测试 PostgreSQL 连接
  - 在用户确认后保存 `runtime.json`

## 会话模型

- `assistant_conversations`
- `assistant_messages`
- `assistant_context_snapshots`
- `assistant_tool_calls`
- `assistant_llm_calls`

服务端在每轮调用时会发送：

- 系统提示词
- 当前部署上下文快照
- 会话摘要
- 最近几轮消息

## 工具约束

允许工具：

- `inspect_bootstrap_status`
- `inspect_runtime_config`
- `test_sqlite_path`
- `test_postgres_connection`
- `save_runtime_config`
- `bootstrap_create_admin`

禁止：

- 热重启服务
- 任意 shell / SQL 执行
- 修改评分、报表或教学业务数据

## 敏感信息处理

- 对话消息、上下文快照、工具调用请求和 LLM 调用日志都做脱敏。
- `database_url` 在助手持久化对象中只保留脱敏版本。
- 真正的 `runtime.json` 仍可保存完整 `database_url`，但需要通过确认工具动作显式写入。

## API

Bootstrap：

- `POST /api/v1/bootstrap/assistant/conversations`
- `GET /api/v1/bootstrap/assistant/conversations/{conversationID}`
- `POST /api/v1/bootstrap/assistant/conversations/{conversationID}/messages`
- `POST /api/v1/bootstrap/assistant/tool-calls/{toolCallID}/confirm`

Admin：

- `POST /api/v1/admin/deployment-assistant/conversations`
- `GET /api/v1/admin/deployment-assistant/conversations/{conversationID}`
- `POST /api/v1/admin/deployment-assistant/conversations/{conversationID}/messages`
- `POST /api/v1/admin/deployment-assistant/tool-calls/{toolCallID}/confirm`

## 当前实现边界

- 已支持服务端持久会话、上下文快照和受控工具确认。
- 已支持无 LLM 配置时的规则化 fallback 回复。
- 认证主链路现在优先走 session cookie，部署助手的 admin 作用域也随之受保护。
- 还未实现：
  - 多步骤正式登录体系
  - 通用知识问答
  - 报表问答助手
  - 评分解释助手
  - 流式输出
