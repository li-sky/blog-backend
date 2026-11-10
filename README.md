# blog-backend

[![CI](https://github.com/li-sky/blog-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/li-sky/blog-backend/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/li-sky/blog-backend/graph/badge.svg)](https://codecov.io/gh/li-sky/blog-backend)

Go + Gin + GORM 的博客后端服务。

## ✨ 特性

- 🚀 RESTful API 设计
- 👤 用户认证与权限管理（JWT）
- 📝 文章管理（CRUD、发布/草稿、标签）
- 💬 评论系统（嵌套回复）
- 🖼️ 图床功能（原图 + 600/1200px 静态 WebP）
- 📰 RSS/Atom Feed 订阅
- ⚙️ 系统设置管理
- 🔐 基于角色的访问控制（RBAC）

## 🏗️ 技术栈

- **语言**: Go 1.25+
- **框架**: Gin (Web), GORM (ORM)
- **数据库**: PostgreSQL
- **认证**: JWT
- **图片处理**: 浏览器 WebAssembly 编码 + Nginx 静态服务
- **容器化**: Docker + Docker Compose

## 🚀 快速开始

### 使用 Docker Compose（推荐）

```bash
# 克隆仓库
git clone https://github.com/your-username/blog-backend.git
cd blog-backend

# 复制环境变量配置
cp .env.example .env

# 编辑 .env 文件（必须修改 JWT_SECRET 和 DB_PASSWORD）
nano .env

# 启动所有服务（API + PostgreSQL + Nginx）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 访问 API（通过 Nginx 代理）
curl http://localhost/api/v1/health
```

健康接口 `GET /health` 和 `GET /api/v1/health` 不要求登录：数据库连接正常时返回
`200 {"status":"ok"}`，连接失败或检查超时时返回 `503 {"status":"unavailable"}`。
容器使用 `/app/blog-server healthcheck` 发起限时本机 GET；探测无需 shell、wget 或 curl，
适用于 distroless 运行镜像。该子命令不会初始化数据库、运行迁移或启动第二个服务。
生产 Compose 可通过 `BLOG_API_IMAGE` 固定已验证的镜像，默认仍使用 GHCR 的 `latest`。

**开发环境架构**：
```
http://localhost        ← Nginx (反向代理)
  ├─ /api/*             → API 服务 (8080)
  └─ /images/*          → /uploads (静态图片)
```

**图片使用示例**：
```
原图:     http://localhost/images/abc123.jpg
600px:    http://localhost/images/abc123_w600.webp
1200px:   http://localhost/images/abc123_w1200.webp
完整尺寸: http://localhost/images/abc123_full.webp
```

### 本地开发（不使用 Docker）

```bash
# 安装 Go 1.25+
go version

# 安装依赖
go mod download

# 启动 PostgreSQL (可以用 Docker)
docker run -d \
  --name blog-postgres \
  -e POSTGRES_USER=blog \
  -e POSTGRES_PASSWORD=blog \
  -e POSTGRES_DB=blog \
  -p 5432:5432 \
  postgres:16-alpine

# 设置环境变量
export DATABASE_URL="postgres://blog:blog@localhost:5432/blog?sslmode=disable"
export JWT_SECRET="your-secret-key"
export STORAGE_FS_BASE_PATH="./uploads"
export STORAGE_FS_BASE_URL="http://localhost:8080/uploads"

# 运行应用
go run main.go

# 或者使用 Air 热重载（推荐开发）
go install github.com/cosmtrek/air@latest
air
```

## 📚 文档

- [API 文档](doc/api_v1.md) - 完整的 API 接口说明
- [部署指南](DEPLOYMENT.md) - 生产环境部署步骤
- [测试用例](test.api.http) - HTTP 请求示例

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test ./... -coverprofile=coverage
go tool cover -html=coverage -o coverage.html

# 查看覆盖率
go tool cover -func coverage
```

## 📦 生产部署

详见 [DEPLOYMENT.md](DEPLOYMENT.md)

**快速部署**：
```bash
# 在 VPS 上
git clone https://github.com/your-username/blog-backend.git
cd blog-backend
cp .env.example .env
nano .env  # 修改配置
chmod +x deploy.sh
./deploy.sh
```

**配置 Nginx** (生产环境推荐):
- 参考 [nginx.prod.conf.example](nginx.prod.conf.example)
- 配置独立图片域名 (`image.yourdomain.com`)
- 配置 SSL 证书 (Let's Encrypt)

## 🖼️ 图片服务

编辑器会在 Web Worker 中通过 WebAssembly 为每张静态图片生成三份 WebP。
原图小于目标宽度时不会放大，但仍使用固定 variant 文件名。动画 GIF
保留原图。历史图片可在管理后台的 `Upgrade Images` 页面四张并发升级。

**响应式图片示例**：
```html
<picture>
  <source srcset="https://image.yourdomain.com/photo_w600.webp"
          type="image/webp" media="(max-width: 768px)">
  <source srcset="https://image.yourdomain.com/photo_w1200.webp"
          type="image/webp" media="(min-width: 769px)">
  <img src="https://image.yourdomain.com/photo.jpg" alt="photo">
</picture>
```

## 🔧 环境变量

关键环境变量说明：

```bash
# 数据库
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable

# JWT 认证
JWT_SECRET=your-very-long-random-secret-key

# 图片存储
STORAGE_BACKEND=filesystem
STORAGE_FS_BASE_PATH=/uploads                    # 存储路径
STORAGE_FS_BASE_URL=https://image.yourdomain.com  # 访问 URL
```

完整配置见 [.env.example](.env.example)

## 📝 许可证

MIT

> 说明：覆盖率 badge 需要启用 Codecov（如果仓库是私有仓库，请在 GitHub Secrets 配置 `CODECOV_TOKEN`）。
