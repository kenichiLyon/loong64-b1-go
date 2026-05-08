# 认证与会话

当前主链路认证采用最小 session 基线：

- `username + password`
- 服务端创建 `auth_sessions`
- 浏览器持有 `httpOnly` cookie
- 业务 API 通过服务端解析当前 actor

## API

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `PUT /api/v1/auth/password`
- `GET /api/v1/me`
- `PUT /api/v1/admin/users/{userID}/password`

## 数据对象

- `users.password_hash`
- `auth_sessions`

`auth_sessions` 保存：

- `id`
- `user_id`
- `token_hash`
- `expires_at`
- `created_at`
- `last_seen_at`

数据库中不保存原始 session token，只保存其 SHA-256。

## Cookie 策略

- `HttpOnly=true`
- `SameSite=Lax`
- `Secure` 由 `SESSION_SECURE_COOKIE` 控制，生产建议开启

相关环境变量：

```env
SESSION_COOKIE_NAME=loong64_b1_session
CSRF_COOKIE_NAME=loong64_b1_csrf
SESSION_TTL=168h
SESSION_CLEANUP_INTERVAL=1h
SESSION_SECURE_COOKIE=false
```

## CSRF 保护

- 对所有依赖 session cookie 的变更请求，服务端要求：
  - session cookie
  - CSRF cookie
  - `X-CSRF-Token` 请求头
- 校验方式是双提交 cookie：`X-CSRF-Token` 必须与 `CSRF_COOKIE_NAME` cookie 的值一致。
- 主要影响：
  - `POST /api/v1/auth/logout`
  - `POST /api/v1/admin/*`
  - `PUT /api/v1/admin/*`
  - 所有已初始化后的 teacher / student / admin 写接口
- bootstrap 首次初始化接口不依赖已有 session，因此不走这套校验。

## Bootstrap 行为

- `POST /api/v1/bootstrap/admin` 创建首个管理员后会直接签发 session cookie 和 CSRF cookie
- bootstrap assistant 中确认 `bootstrap_create_admin` 也会直接签发 session cookie 和 CSRF cookie

## 管理员密码重置

- 管理员可以为已有用户设置或重置密码
- 密码更新后立即生效
- 管理员重置目标用户密码时，会同步吊销该用户现有全部 session

## 自助改密

- 已登录用户可调用 `PUT /api/v1/auth/password`
- 请求体要求：
  - `current_password`
  - `new_password`
- 修改成功后，服务端会：
  - 更新 `users.password_hash`
  - 吊销该用户当前和其他全部 session
  - 清除当前浏览器中的 session / csrf cookie
- 因此前端必须提示用户重新登录

## 会话清理

- 过期 `auth_sessions` 不依赖独立 worker；当前实现采用认证流量触发的机会式清理。
- `SESSION_CLEANUP_INTERVAL` 控制清理最小间隔，默认 `1h`。
- 每次登录或解析 session 时，服务端最多每个间隔执行一次：
  - `DELETE FROM auth_sessions WHERE expires_at <= now()`
- 设置为 `0` 或非正值时可关闭该清理策略。

## 开发态 bypass

`X-Actor-ID / X-Actor-Roles` 没有被完全删除，但只应在以下条件下使用：

- `DEV_AUTH_BYPASS=true`
- `APP_ENV!=production`
- 请求来自 localhost

除此之外，主链路必须走 session cookie。
