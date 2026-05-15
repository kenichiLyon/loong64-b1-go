# Python AI Worker（当前 python-ai-gateway）

本文件说明当前项目里的 Python AI Worker 负责什么、如何本地调试，以及为什么它是正式组成部分而不是临时脚本。当前目录、包名、环境变量前缀和 systemd 单元仍沿用 `python-ai-gateway` 兼容命名；后续升级应向更通用的 AI Worker 形态收敛。

## 1. 为什么引入 Python

引入 Python 不是为了替换 Go，而是为了补齐 Go 在 AI 生态上的客观短板。

当前明确交给 Python 的能力包括：

- 文档解析增强
- 本地模型 / 校内模型网关接入
- 检索 / RAG
- 结构化输出清洗
- 推理上下文组装

Go 主服务继续负责：

- 对外 API
- 登录 / 权限 / 会话
- 教学业务流程
- 数据库和对象存储写入
- 审计日志
- 教师复核与发布

一句话：

- `Go 管业务`
- `Python AI Worker 管推理`

## 2. 当前职责

Python AI Worker 当前提供这些内部接口：

- `GET /health/live`
- `GET /health/ready`
- `POST /internal/parse-artifact`
- `POST /internal/evaluate-submission`
- `POST /internal/build-retrieval-index`
- `POST /internal/query-retrieval`

已经具备的能力：

- 多格式附件解析
- 初评请求组装与模型调用
- 检索索引构建
- 检索查询
- retrieval-augmented prompt
- 细粒度 evidence refs 对齐

当前仍然属于后续增强的部分：

- OCR 深化
- 持久化向量检索
- embedding 生成链路
- 多 provider 路由
- 本地模型运维自动化

## 3. 当前目录

当前实现仍放在 `python-ai-gateway/` 下，暂不在部署文档中改名，以避免服务名和发布包路径漂移：

- `python-ai-gateway/ai_gateway/app.py`
- `python-ai-gateway/ai_gateway/models.py`
- `python-ai-gateway/ai_gateway/parser.py`
- `python-ai-gateway/ai_gateway/evaluator.py`
- `python-ai-gateway/ai_gateway/retrieval.py`
- `python-ai-gateway/pyproject.toml`
- `python-ai-gateway/requirements.txt`

## 4. 本地运行

Linux / macOS：

```bash
cd python-ai-gateway
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Windows PowerShell：

```powershell
cd python-ai-gateway
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

## 5. 调试与依赖

当前默认、可复现的安装方式是：

```bash
pip install -r requirements.txt
```

不要求引入 `uv` 这类额外包管理器。

推荐的最小语法验证命令：

```bash
pip install -r python-ai-gateway/requirements.txt
python -m py_compile python-ai-gateway/ai_gateway/app.py python-ai-gateway/ai_gateway/models.py python-ai-gateway/ai_gateway/parser.py python-ai-gateway/ai_gateway/evaluator.py python-ai-gateway/ai_gateway/retrieval.py
```

推荐调试顺序：

1. 先验证 `app.py` 能启动
2. 再验证 `parser.py`
3. 再验证 `retrieval.py`
4. 最后验证 `evaluator.py`

## 6. Go 侧接入

Go 侧核心环境变量：

```env
AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_TIMEOUT=10s
AI_GATEWAY_API_KEY=
```

Python 侧模型环境变量：

```env
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

当 `AI_GATEWAY_BASE_URL` 被配置时，Go 会：

- 在 readiness 检查中纳入 Python AI Worker
- 优先把解析和初评交给 Python AI Worker
- 必要时回退到 Go 侧已有能力

## 7. 部署建议

当前正式推荐部署形态是双服务：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

服务名暂时保留 `python-ai-gateway.service`，语义上按 AI Worker 维护。

部署建议：

- Python 使用独立 venv
- 使用 systemd 管理
- 与 Go 主服务同机部署优先
- 模型服务可同机或走校内 / 云端网关

## 8. 当前限制

当前检索仍然是：

- 有界内存索引

这不是最终形态，但它的价值在于：

- 先把 Go / Python 边界跑通
- 不在 LoongArch 交付前过早引入外部检索基础设施

后续如果升级为 embedding / 向量库，也应保持现有内部接口尽量不变。

## 9. 当前结论

现在这套 Python AI Worker 不是可选实验脚本，而是：

- `项目 AI 能力层的正式组成部分`

后续所有本地模型、检索、RAG、解析增强，优先都应沿 AI Worker 线继续演进，而不是重新塞回 Go 主服务。
