# 安全与会话验证记录

日期：

- `2026-05-14`

范围：

- session 登录 / 登出
- CSRF 双提交 cookie 校验
- 同源 `Origin` 限制
- 自助改密后旧 session 失效
- 管理员重置密码后旧 session 失效

说明：

- 本记录基于当前仓库中的自动化测试与本地运行结果整理
- 这份记录用于支撑“本地可交付候选”判断
- 它不能替代目标机上的最终安全验收

## 1. 已验证命令

```bash
go test ./internal/authn ./internal/api
```

## 2. 覆盖的关键用例

### 2.1 登录、解析与登出

文件：

- `internal/authn/service_test.go`
- `internal/api/auth_test.go`

已验证：

- 正确账号密码可登录并创建 session
- 请求可通过 session cookie 解析当前用户
- 登出后原 session 立即失效

代表性用例：

- `TestLoginResolveAndLogout`
- `TestLoginFlowAndLogoutClearsSession`

### 2.2 自助改密后旧 session 失效

文件：

- `internal/authn/service_test.go`
- `internal/api/auth_test.go`

已验证：

- 用户自助改密后，当前与其他旧 session 全部失效
- 旧密码再次登录失败
- 新密码登录成功

代表性用例：

- `TestChangePasswordRevokesAllSessions`
- `TestAuthChangePasswordRevokesSessionAndOldPassword`

### 2.3 管理员重置密码后目标用户旧 session 失效

文件：

- `internal/api/auth_test.go`

已验证：

- 管理员调用 `PUT /api/v1/admin/users/{userID}/password`
- 目标用户旧 session 被吊销
- 旧密码登录失败
- 新密码登录成功

代表性用例：

- `TestAdminPasswordResetRevokesStudentSessions`

### 2.4 CSRF 缺失时拒绝写请求

文件：

- `internal/api/auth_test.go`
- `internal/authn/service_test.go`

已验证：

- 基于 session 的写请求如果缺少 `X-CSRF-Token`，返回 `403`
- CSRF 双提交 cookie 校验路径正常工作

代表性用例：

- `TestAuthRejectsMutatingRequestWithoutCSRFFromSession`
- `TestValidateCSRFAcceptsSameOriginAndRejectsCrossOrigin`

### 2.5 跨站 Origin 写请求拒绝

文件：

- `internal/api/auth_test.go`
- `internal/authn/service_test.go`

已验证：

- 如果 `Origin` 与当前 host 不一致，写请求返回 `403`

代表性用例：

- `TestAuthRejectsMutatingRequestWithCrossOrigin`
- `TestValidateCSRFAcceptsSameOriginAndRejectsCrossOrigin`

### 2.6 Session 续期与过期清理

文件：

- `internal/authn/service_test.go`

已验证：

- session 到达刷新间隔后会续期并重写 cookie
- 过期 session 可被清理，并在后续请求中失效

代表性用例：

- `TestRefreshSessionIfDueExtendsExpiryAndRewritesCookies`
- `TestCleanupExpiredSessionsDeletesExpiredRows`

## 3. 本地验证结论

当前本地可确认：

- session 登录 / 登出逻辑正常
- CSRF 双提交 cookie 保护存在且已测试
- cross-origin 写请求拒绝逻辑存在且已测试
- 自助改密后旧 session 失效逻辑存在且已测试
- 管理员重置密码后旧 session 失效逻辑存在且已测试

## 4. 仍需目标环境复验的内容

以下内容仍建议在 `LoongArch + 银河麒麟` 目标环境复验并留档：

- 浏览器真实 cookie 行为
- 反向代理 / 域名部署下的 `Origin` / `Referer` 行为
- HTTPS / secure cookie 配置
- systemd 长时间运行下的 session 清理与续期行为

## 5. 当前结论

结论：

- `本地安全与会话验证已具备可交付候选证据`

但在正式交付判断上，仍然缺少：

- `LoongArch + 银河麒麟目标环境复验留档`
