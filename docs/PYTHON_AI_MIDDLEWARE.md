# Python 推理微服务

本文件说明当前项目里的 Python 服务到底是做什么的、怎么调试、以及为什么它必须存在。

## 1. 为什么引入 Python

引入 Python 的原因不是“换技术栈”，而是明确补 Go 在 AI 生态上的不足。

当前我们确认需要 Python 来承接这些能力：

- 文档解析增强
- 本地大模型 / 校内模型网关接入
- 检索 / RAG
- 结构化输出清洗
- 推理上下文组装

Go 主服务继续负责：

- 对外 API
- 用户 / 权限 / 会话
- 教学业务流程
- 数据库和对象存储写入
- 审计日志
- 教师复核与发布

所以边界是：

- `Go 管业务`
- `Python 管推理`

## 2. 当前职责

当前 Python 微服务负责这些内部接口：
# Python 推理微服务

本文件说明当前项目里的 Python 服务到底是做什么的、怎么调试、以及为什么它必须存在。

## 1. 为什么引入 Python

引入 Python 的原因不是“换技术栈”，而是明确补 Go 在 AI 生态上的不足。

当前我们确认需要 Python 来承接这些能力：

- 文档解析增强
- 本地大模型 / 校内模型网关接入
- 检索 / RAG
- 结构化输出清洗
- 推理上下文组装

Go 主服务继续负责：

- 对外 API
- 用户 / 权限 / 会话
- 教学业务流程
- 数据库和对象存储写入
- 审计日志
- 教师复核与发布

所以边界是：

- `Go 管业务`
- `Python 管推理`

## 2. 当前职责

当前 Python 微服务负责这些内部接口：

- `GET /health/live`
- `GET /health/ready`
- `POST /internal/parse-artifact`
- `POST /internal/evaluate-submission`
- `POST /internal/build-retrieval-index`
- `POST /internal/query-retrieval`

当前已经具备的真实能力：

- 多格式附件解析
- 初评调用
- 检索索引构建
- 检索查询
- retrieval-augmented prompt
- 细粒度 evidence refs 对齐

当前暂未做成完整产品能力的部分：

- OCR 深化
- 持久化向量检索
- embedding 生成链路
- 多 provider 策略路由
- 本地模型运维自动化

## 3. 当前目录

主要文件：

- `python-ai-gateway/ai_gateway/app.py`
- `python-ai-gateway/ai_gateway/models.py`
- `python-ai-gateway/ai_gateway/parser.py`
- `python-ai-gateway/ai_gateway/evaluator.py`
- `python-ai-gateway/ai_gateway/retrieval.py`
- `python-ai-gateway/pyproject.toml`

## 4. 本地运行

Linux / macOS：

```bash
cd python-ai-gateway
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

Windows PowerShell:
Windows PowerShell:

```powershell
cd python-ai-gateway
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
uvicorn ai_gateway.app:app --host 127.0.0.1 --port 8081
```

## 5. 本地调试

最小语法验证：

```bash
python -m py_compile python-ai-gateway/ai_gateway/app.py python-ai-gateway/ai_gateway/models.py python-ai-gateway/ai_gateway/parser.py python-ai-gateway/ai_gateway/evaluator.py python-ai-gateway/ai_gateway/retrieval.py
```

推荐调试顺序：

1. 先验证 `app.py` 能启动
2. 再单看 `parser.py`
3. 再单看 `retrieval.py`
4. 最后看 `evaluator.py`

如果本地没有模型服务，也可以只验证：

- 解析接口
- 检索接口
- 结构化请求 / 响应模型

## 6. Go 侧接入

Go 侧环境变量：

```env
AI_GATEWAY_BASE_URL=http://127.0.0.1:8081
AI_GATEWAY_TIMEOUT=10s
AI_GATEWAY_API_KEY=
```

Python 侧环境变量：
Python 侧环境变量：

```env
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

当 `AI_GATEWAY_BASE_URL` 被配置时，Go 会：

- readiness 检查中纳入 Python 服务
- 优先把解析和初评交给 Python 微服务
- 在必要时回退到 Go 侧已有能力

## 7. 部署建议

当前推荐部署是双服务：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

Python 不建议嵌进 Go 进程，也不建议把主业务迁移到 Python。

生产部署建议：

- Python 使用独立 venv
- systemd 管理
- 与 Go 主服务同机部署优先
- 模型服务可以同机，也可以走校内网关 / 云端网关

## 8. 当前限制

当前检索仍是：

- 有界内存索引

这不是最终形态，但它有两个优点：

- 先把边界跑通
- 不在 LoongArch 交付前过早引入外部检索基础设施

后续如果升级为 embedding / 向量库，也应该保持现有内部接口不变。

## 9. 当前结论

现在这套 Python 服务不是“可有可无”的实验脚本，而是：

- `项目 AI 能力层的正式组成部分`

后续所有本地模型、检索、RAG、解析增强，优先都应该沿这条线继续做，而不是重新塞回 Go 主服务。 
AI_GATEWAY_API_KEY=
AI_GATEWAY_LLM_BASE_URL=http://127.0.0.1:8000/v1
AI_GATEWAY_LLM_API_KEY=
AI_GATEWAY_LLM_MODEL=local-model
AI_GATEWAY_LLM_TIMEOUT=30
```

当 `AI_GATEWAY_BASE_URL` 被配置时，Go 会：

- readiness 检查中纳入 Python 服务
- 优先把解析和初评交给 Python 微服务
- 在必要时回退到 Go 侧已有能力

## 7. 部署建议

当前推荐部署是双服务：

1. `loong64-b1-go.service`
2. `python-ai-gateway.service`

Python 不建议嵌进 Go 进程，也不建议把主业务迁移到 Python。

生产部署建议：

- Python 使用独立 venv
- systemd 管理
- 与 Go 主服务同机部署优先
- 模型服务可以同机，也可以走校内网关 / 云端网关

## 8. 当前限制

当前检索仍是：

- 有界内存索引

这不是最终形态，但它有两个优点：

- 先把边界跑通
- 不在 LoongArch 交付前过早引入外部检索基础设施

后续如果升级为 embedding / 向量库，也应该保持现有内部接口不变。

## 9. 当前结论

现在这套 Python 服务不是“可有可无”的实验脚本，而是：

- `项目 AI 能力层的正式组成部分`

后续所有本地模型、检索、RAG、解析增强，优先都应该沿这条线继续做，而不是重新塞回 Go 主服务。 
