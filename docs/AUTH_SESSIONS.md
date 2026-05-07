# 认证与会话

当前主链路认证采用最小 session 基线：

- `username + password`
- 服务端创建 `auth_sessions`
- 浏览器持有 `httpOnly` cookie
- 业务 API 通过服务端解析当前 actor

## API

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/me`

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
SESSION_TTL=168h
SESSION_SECURE_COOKIE=false
```

## Bootstrap 行为

- `POST /api/v1/bootstrap/admin` 创建首个管理员后会直接签发 session cookie
- bootstrap assistant 中确认 `bootstrap_create_admin` 也会直接签发 session cookie

## 开发态 bypass

`X-Actor-ID / X-Actor-Roles` 没有被完全删除，但只应在以下条件下使用：

- `DEV_AUTH_BYPASS=true`
- `APP_ENV!=production`
- 请求来自 localhost

除此之外，主链路必须走 session cookie。
