# Web 前端计划

首版前端采用 Vue 3 + Vite + TypeScript，提供 PC Web 工作台。当前 Stage 5.5 已建立可运行 MVP，用于演示学生提交、教师核查初评、教师复核发布和学生查看已发布评价的主流程。

## 功能范围

- 学生：创建提交、上传成果、登记 Git 链接、查看提交详情、查看已发布评价。
- 教师：查看实验提交列表、查看提交详情、运行规则/LLM 初评、保存复核草稿、发布最终评价。
- 管理员：本阶段暂不做完整页面；继续保留后端 API 能力，后续补管理工作台。

## 开发启动

```bash
cd web
npm ci
npm run dev
```

默认 Vite 地址：`http://127.0.0.1:5173`。

开发服务器会把 `/api` 和 `/health` 代理到 `http://127.0.0.1:8080`。如需固定 API 地址，可设置：

```bash
VITE_API_BASE_URL=http://127.0.0.1:8080
```

## 构建

```bash
cd web
npm run lint
npm run build
```

构建产物输出到 `web/dist`，目标部署可由 Go 服务前置 Nginx 或银河麒麟 systemd 环境中的静态资源服务托管。

## 登录与开发态身份

当前主链路默认使用服务端 session：

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/me`
- `PUT /api/v1/admin/users/{userID}/password`

仅在本机调试并启用 `DEV_AUTH_BYPASS=true` 时，才建议继续使用：

- `X-Actor-ID`
- `X-Actor-Roles`

生产环境仍应继续沿着服务端会话或统一认证网关演进。

## 管理员能力

当前前端已包含：

- bootstrap 创建首个管理员
- 运行配置保存
- 部署助手
- 管理员为现有用户设置/重置密码

## LoongArch 注意事项

前端构建在 amd64 开发/CI 环境完成，目标 LoongArch + 银河麒麟只托管静态文件，不要求在目标机安装 Node.js。若必须在目标机重新构建，需要单独验证 Node/Vite 在 LoongArch 上的可用性。
