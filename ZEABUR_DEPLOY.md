# Zeabur 部署指南

本指南将帮助你将 Sub2API 部署到 Zeabur 平台。

## 前置条件

- Zeabur 账号（访问 https://zeabur.com 注册）
- GitHub 账号（用于连接代码仓库）

## 部署步骤

### 1. 准备代码仓库

确保代码已推送到 GitHub：

```bash
cd D:\my_code\工具\sub2api
git add .
git commit -m "准备部署到 Zeabur"
git push origin main
```

### 2. 在 Zeabur 创建项目

1. 登录 [Zeabur Dashboard](https://dash.zeabur.com)
2. 点击 **Create Project**
3. 输入项目名称（例如：sub2api）

### 3. 添加 PostgreSQL 数据库

1. 在项目页面点击 **Add Service**
2. 选择 **Database**
3. 选择 **PostgreSQL**
4. 等待数据库部署完成

### 4. 添加 Redis 缓存

1. 再次点击 **Add Service**
2. 选择 **Database**
3. 选择 **Redis**
4. 等待 Redis 部署完成

### 5. 部署 Sub2API 应用

1. 点击 **Add Service**
2. 选择 **GitHub**
3. 授权并选择 `Wei-Shaw/sub2api` 仓库
4. Zeabur 会自动检测 Dockerfile 并开始构建

### 6. 配置环境变量

在 Sub2API 服务的设置页面，添加以下环境变量：

#### 必需的环境变量

```bash
# JWT 密钥（必须修改为随机字符串）
JWT_SECRET=your-super-secret-jwt-key-at-least-32-chars

# 服务器配置
SERVER_PORT=8080
SERVER_MODE=release
```

#### 数据库连接（Zeabur 会自动配置）

Zeabur 会自动注入以下环境变量，无需手动配置：
- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DATABASE`
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_PASSWORD`

但是 Sub2API 使用不同的变量名，你需要手动映射：

```bash
DATABASE_HOST=${POSTGRES_HOST}
DATABASE_PORT=${POSTGRES_PORT}
DATABASE_USER=${POSTGRES_USER}
DATABASE_PASSWORD=${POSTGRES_PASSWORD}
DATABASE_DBNAME=${POSTGRES_DATABASE}
REDIS_HOST=${REDIS_HOST}
REDIS_PORT=${REDIS_PORT}
REDIS_PASSWORD=${REDIS_PASSWORD}
```

#### 可选环境变量

```bash
# 管理员账号（首次启动时创建）
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your-admin-password

# 默认配置
DEFAULT_USER_CONCURRENCY=5
DEFAULT_USER_BALANCE=0
DEFAULT_API_KEY_PREFIX=sk-

# CORS（替换为你的 Zeabur 域名）
CORS_ALLOWED_ORIGINS=https://your-app.zeabur.app

# 安全配置
SECURITY_URL_ALLOWLIST_ENABLED=true
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=false
```

### 7. 配置域名（可选）

1. 在服务设置中找到 **Networking** 或 **Domain** 选项
2. Zeabur 会自动分配一个 `.zeabur.app` 域名
3. 或者绑定你自己的域名

### 8. 访问应用

1. 等待部署完成
2. 访问 Zeabur 分配的域名（例如：`https://sub2api-xxx.zeabur.app`）
3. 使用管理员账号登录

## 环境变量配置说明

### 数据库变量映射

Sub2API 期望的变量名与 Zeabur 注入的变量名不同，需要手动映射：

| Sub2API 期望 | Zeabur 提供 |
|-------------|------------|
| `DATABASE_HOST` | `POSTGRES_HOST` |
| `DATABASE_PORT` | `POSTGRES_PORT` |
| `DATABASE_USER` | `POSTGRES_USER` |
| `DATABASE_PASSWORD` | `POSTGRES_PASSWORD` |
| `DATABASE_DBNAME` | `POSTGRES_DATABASE` |

### Redis 变量映射

| Sub2API 期望 | Zeabur 提供 |
|-------------|------------|
| `REDIS_HOST` | `REDIS_HOST` |
| `REDIS_PORT` | `REDIS_PORT` |
| `REDIS_PASSWORD` | `REDIS_PASSWORD` |

## 故障排查

### 应用无法启动

1. 检查日志：在 Zeabur 服务页面查看 **Logs** 标签
2. 确认所有环境变量已正确设置
3. 确认 PostgreSQL 和 Redis 服务已正常运行

### 数据库连接失败

1. 检查环境变量映射是否正确
2. 确认 PostgreSQL 服务状态
3. 查看数据库连接日志

### 无法访问管理后台

1. 确认应用已成功启动
2. 检查域名配置
3. 确认端口设置为 `8080`

## 更新应用

Zeabur 支持自动部署：

1. 推送代码到 GitHub：`git push origin main`
2. Zeabur 会自动检测并重新构建部署

## 重要提示

1. **修改 JWT_SECRET**：默认值不安全，必须修改为随机字符串
2. **配置 CORS**：将 `CORS_ALLOWED_ORIGINS` 设置为你的实际域名
3. **数据持久化**：Zeabur 的 PostgreSQL 和 Redis 数据会自动持久化
4. **健康检查**：Dockerfile 中已配置健康检查端点 `/health`

## 成本优化

- Zeabur 提供免费套餐，适合测试和小规模使用
- 生产环境建议升级到付费套餐以获得更好的性能和可用性

## 相关链接

- [Zeabur 文档](https://zeabur.com/docs)
- [Sub2API 项目文档](https://github.com/Wei-Shaw/sub2api)
