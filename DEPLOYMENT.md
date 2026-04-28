# 部署指南

## 使用 Docker 镜像部署到 VPS

### 前置条件

- 一台 VPS（Linux 系统）
- 已安装 Docker 和 Docker Compose
- 域名（可选，用于 HTTPS）

---

## 方式一：使用 GitHub Actions 自动构建（推荐）

### 1. 配置 GitHub Repository

1. 推送代码到 GitHub
2. GitHub Actions 会自动构建并推送镜像到 `ghcr.io`
3. 在 VPS 上只需要拉取镜像即可

### 2. VPS 部署步骤

```bash
# 克隆仓库（只需要配置文件）
git clone https://github.com/your-username/blog-backend.git
cd blog-backend

# 复制环境变量模板
cp .env.example .env

# 编辑环境变量（重要！）
nano .env
```

修改以下内容：
- `DB_PASSWORD`: 设置强密码
- `JWT_SECRET`: 生成随机密钥（可用 `openssl rand -base64 64`）

**图床存储配置**（可选）：
- `STORAGE_BACKEND`: 存储后端类型，默认 `filesystem`
- `STORAGE_FS_BASE_PATH`: 文件存储基础路径，默认 `./uploads`
- `STORAGE_FS_BASE_URL`: 文件访问完整 URL 前缀（如 `https://image.yourdomain.com`）

**注意事项**：
1. 使用文件系统存储时，需要确保 `STORAGE_FS_BASE_PATH` 目录存在且容器有写入权限
2. 生产环境建议挂载数据卷保存上传的文件，避免容器重启后文件丢失
3. **推荐配置独立图片域名**（如 `image.yourdomain.com`），便于未来迁移到 CDN 或对象存储
4. 如使用 Nginx 提供静态文件服务，需要配置对应的 server 块（见下方 Nginx 配置）
5. 使用完整 URL 的好处：切换存储后端时只需修改环境变量，前端无需改动

```bash
# 执行部署脚本
chmod +x deploy.sh
./deploy.sh
```

### 3. Docker Compose 服务说明

**生产环境服务架构**：

```
┌─────────────┐     ┌─────────────┐
│   Nginx     │────▶│  imgproxy   │ (图片转换)
│  (反向代理) │     └─────────────┘
└──────┬──────┘              │
       │                     │
       │                ┌────▼─────┐
       │                │ uploads  │ (共享卷)
       │                └────▲─────┘
       │                     │
       ▼                     │
┌─────────────┐     ┌────────┴─────┐
│     API     │────▶│  PostgreSQL  │
│  (Go应用)   │     │   (数据库)   │
└─────────────┘     └──────────────┘
```

**服务说明**：

1. **api**: Blog 后端应用
   - 端口: 8080（内部）
   - 负责: API 逻辑、图片上传（保存原图）
   - 数据卷: `uploads-data`（存储上传的图片）

2. **imgproxy**: 图片动态转换服务
   - 端口: 8080（内部）
   - 负责: 实时转换图片尺寸和格式
   - 数据卷: `uploads-data`（只读）

3. **db**: PostgreSQL 数据库
   - 端口: 5432（内部）
   - 数据卷: `db-data`（持久化数据）

4. **nginx**（可选，VPS 上需要独立配置）:
   - 端口: 80/443
   - 负责: 反向代理、SSL 终止、缓存

**环境变量配置**：

在 `.env` 文件中添加：
```bash
# 图片访问 URL（使用图片域名）
STORAGE_FS_BASE_URL=https://image.yourdomain.com

# 图片存储路径（容器内路径）
STORAGE_FS_BASE_PATH=/uploads
```

### 4. 验证部署

```bash
# 查看服务状态
docker-compose -f docker-compose.prod.yml ps

# 查看日志
docker-compose -f docker-compose.prod.yml logs -f api

# 测试 API
curl http://localhost:8080/health

# 测试 imgproxy（如果端口暴露）
# curl http://localhost:8081/health
```

---

## 方式二：手动构建并推送镜像

### 1. 本地构建镜像

```powershell
# Windows PowerShell
# 登录 GitHub Container Registry
$env:CR_PAT="your_github_token"
echo $env:CR_PAT | docker login ghcr.io -u your-username --password-stdin

# 构建镜像
docker build -t ghcr.io/your-username/blog-backend:latest .

# 推送镜像
docker push ghcr.io/your-username/blog-backend:latest
```

### 2. 在 VPS 上拉取并运行

```bash
# 登录镜像仓库（如果是私有镜像）
docker login ghcr.io -u your-username

# 拉取镜像
docker pull ghcr.io/your-username/blog-backend:latest

# 启动服务
docker-compose -f docker-compose.prod.yml up -d
```

---

## 方式三：使用其他镜像仓库

### Docker Hub

```bash
# 登录
docker login

# 构建并推送
docker build -t your-username/blog-backend:latest .
docker push your-username/blog-backend:latest
```

修改 `docker-compose.prod.yml` 中的镜像地址：
```yaml
image: your-username/blog-backend:latest
```

### 阿里云容器镜像服务

```bash
# 登录
docker login --username=your-aliyun-account registry.cn-hangzhou.aliyuncs.com

# 构建并推送
docker build -t registry.cn-hangzhou.aliyuncs.com/your-namespace/blog-backend:latest .
docker push registry.cn-hangzhou.aliyuncs.com/your-namespace/blog-backend:latest
```

---

## 配置 Nginx 反向代理（推荐）

### 1. 安装 Nginx

```bash
sudo apt update
sudo apt install nginx -y
```

### 2. 创建配置文件

```bash
sudo nano /etc/nginx/sites-available/blog-api
```

添加配置：

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    client_max_body_size 50M;

    # API 代理
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        proxy_cache_bypass $http_upgrade;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

}
```

**方案A：使用独立图片域名 + Nginx 动态转换（推荐生产环境）**

支持动态调整分辨率和格式转换，提供更好的移动端加载体验。

**选项 1：使用 Nginx image_filter 模块**

创建图片服务器配置：

```bash
sudo nano /etc/nginx/sites-available/blog-images
```

```nginx
# 图片动态转换服务器
proxy_cache_path /var/cache/nginx/images levels=1:2 keys_zone=images_cache:10m max_size=1g inactive=30d use_temp_path=off;

server {
    listen 80;
    server_name image.yourdomain.com;

    set $upload_root /path/to/your/uploads;  # 修改为实际的上传文件路径

    # 动态图片转换 + 缓存
    location ~ ^/(.+)$ {
        set $filename $1;
        set $width $arg_w;
        set $format $arg_format;

        # 缓存配置
        proxy_cache images_cache;
        proxy_cache_key "$uri$is_args$args";
        proxy_cache_valid 200 30d;
        proxy_cache_valid 404 1m;
        add_header X-Cache-Status $upstream_cache_status;

        # 处理格式转换和缩放
        if ($width) {
            rewrite ^ /resize last;
        }
        if ($format = "webp") {
            rewrite ^ /webp last;
        }

        # 直接返回原图
        root $upload_root;
        expires 30d;
        add_header Cache-Control "public, immutable";
        add_header Access-Control-Allow-Origin "*";
        try_files /$filename =404;
    }

    # 缩放处理
    location /resize {
        internal;
        alias $upload_root;
        image_filter resize $width -;
        image_filter_jpeg_quality 85;
        image_filter_webp_quality 80;
        image_filter_buffer 20M;
        
        expires 30d;
        add_header Cache-Control "public, immutable";
        add_header Access-Control-Allow-Origin "*";
    }

    # WebP 转换
    location /webp {
        internal;
        alias $upload_root;
        image_filter resize $width -;
        image_filter_jpeg_quality 80;
        image_filter_webp_quality 75;
        image_filter_buffer 20M;
        
        # 强制输出 WebP
        add_header Content-Type image/webp;
        
        expires 30d;
        add_header Cache-Control "public, immutable";
        add_header Access-Control-Allow-Origin "*";
    }
}
```

**选项 2：使用 imgproxy（更强大，推荐）**

安装 imgproxy：

```bash
# 使用 Docker 运行 imgproxy
docker run -d \
  --name imgproxy \
  -p 8081:8080 \
  -e IMGPROXY_LOCAL_FILESYSTEM_ROOT=/uploads \
  -e IMGPROXY_ENABLE_WEBP_DETECTION=true \
  -e IMGPROXY_USE_ETAG=true \
  -v /path/to/your/uploads:/uploads:ro \
  darthsim/imgproxy:latest
```

Nginx 配置：

```bash
sudo nano /etc/nginx/sites-available/blog-images
```

```nginx
# imgproxy 代理 + 缓存
proxy_cache_path /var/cache/nginx/images levels=1:2 keys_zone=images_cache:10m max_size=2g inactive=30d use_temp_path=off;

server {
    listen 80;
    server_name image.yourdomain.com;

    location / {
        # 将请求代理到 imgproxy
        set $width $arg_w;
        set $format $arg_format;
        set $imgproxy_url "plain/local:///";

        # 构建 imgproxy URL
        if ($width) {
            set $imgproxy_url "resize:fill:${width}:0:0/plain/local:///";
        }
        if ($format = "webp") {
            set $imgproxy_url "${imgproxy_url}@webp";
        }

        proxy_pass http://localhost:8081/$imgproxy_url$uri;
        
        # 缓存配置
        proxy_cache images_cache;
        proxy_cache_key "$uri$is_args$args";
        proxy_cache_valid 200 30d;
        proxy_cache_valid 404 1m;
        add_header X-Cache-Status $upstream_cache_status;

        # 头部设置
        proxy_set_header Host $host;
        proxy_pass_request_headers on;
        
        add_header Cache-Control "public, max-age=2592000, immutable";
        add_header Access-Control-Allow-Origin "*";
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/blog-images /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# 配置 SSL
sudo certbot --nginx -d image.yourdomain.com
```

然后设置环境变量：
```bash
STORAGE_FS_BASE_URL=https://image.yourdomain.com
```

**使用示例**：
```
# 原图
https://image.yourdomain.com/abc123.jpg

# 640px 宽度
https://image.yourdomain.com/abc123.jpg?w=640

# WebP 格式
https://image.yourdomain.com/abc123.jpg?format=webp

# 组合：640px + WebP
https://image.yourdomain.com/abc123.jpg?w=640&format=webp
```

**方案B：简单静态文件服务（适合小型项目或开发环境）**

如果不需要动态转换功能，可以使用简单的静态文件服务：

```bash
sudo nano /etc/nginx/sites-available/blog-images
```

```nginx
# 图片静态文件服务器
server {
    listen 80;
    server_name image.yourdomain.com;

    root /path/to/your/uploads;  # 修改为实际的上传文件路径
    
    location / {
        expires 30d;
        add_header Cache-Control "public, immutable";
        add_header Access-Control-Allow-Origin "*";
        access_log off;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/blog-images /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# 配置 SSL
sudo certbot --nginx -d image.yourdomain.com
```

然后设置环境变量：
```bash
STORAGE_FS_BASE_URL=https://image.yourdomain.com
```

**方案C：使用同域名子路径**

在 API 配置中添加：

```nginx
    # 静态文件服务（用于访问上传的图片）
    location /uploads/ {
        alias /path/to/your/uploads/;  # 修改为实际的上传文件路径
        expires 7d;
        add_header Cache-Control "public, immutable";
        access_log off;
    }
}
```

然后设置环境变量：
```bash
STORAGE_FS_BASE_URL=https://api.yourdomain.com/uploads
```

**方案对比**：

| 方案 | 优势 | 适用场景 |
|------|------|----------|
| 方案A-选项1 (Nginx image_filter) | ✅ 动态转换<br>✅ 无需额外服务<br>✅ 配置简单 | 中小型项目，需要基本的图片优化 |
| 方案A-选项2 (imgproxy) | ✅ 功能强大<br>✅ 性能优异<br>✅ 支持更多格式和操作 | **生产环境推荐**，高流量项目 |
| 方案B (静态文件) | ✅ 最简单<br>✅ 资源占用低 | 小型项目，图片数量少 |
| 方案C (同域名) | ✅ 无需额外域名 | 开发环境，快速上线 |

**注意事项**：
- `client_max_body_size 50M` 用于允许上传最大 50MB 的文件
- 如使用 Docker，需要将上传目录挂载到宿主机，并在 Nginx 中配置宿主机的实际路径
- **生产环境强烈推荐方案A-选项2（imgproxy）**，提供最佳的移动端加载体验
- Nginx image_filter 需要编译时包含 `--with-http_image_filter_module`，检查：`nginx -V 2>&1 | grep image_filter`
- imgproxy 支持更多高级功能（裁剪、水印、滤镜等），详见：https://docs.imgproxy.net/

### 3. 启用配置

```bash
sudo ln -s /etc/nginx/sites-available/blog-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 4. 配置 SSL（可选）

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx -y

# 获取免费 SSL 证书
sudo certbot --nginx -d api.yourdomain.com

# 自动续期
sudo systemctl enable certbot.timer
```

---

## 常用管理命令

```bash
# 查看服务状态
docker-compose -f docker-compose.prod.yml ps

# 查看日志
docker-compose -f docker-compose.prod.yml logs -f api

# 重启服务
docker-compose -f docker-compose.prod.yml restart

# 停止服务
docker-compose -f docker-compose.prod.yml down

# 更新到最新版本
./deploy.sh

# 备份数据库
docker-compose -f docker-compose.prod.yml exec db pg_dump -U blog blog > backup_$(date +%Y%m%d_%H%M%S).sql

# 恢复数据库
docker-compose -f docker-compose.prod.yml exec -T db psql -U blog blog < backup.sql

# 查看资源使用
docker stats

# 清理未使用的镜像和容器
docker system prune -a
```

---

## 监控和日志

### 查看实时日志

```bash
# 所有服务
docker-compose -f docker-compose.prod.yml logs -f

# 仅 API
docker-compose -f docker-compose.prod.yml logs -f api

# 仅数据库
docker-compose -f docker-compose.prod.yml logs -f db
```

### 进入容器调试

```bash
# 进入 API 容器
docker-compose -f docker-compose.prod.yml exec api sh

# 进入数据库容器
docker-compose -f docker-compose.prod.yml exec db psql -U blog blog
```

---

## 安全建议

1. **强密码**：使用强随机密码
2. **防火墙**：只开放必要端口（80, 443）
3. **定期更新**：保持系统和 Docker 镜像更新
4. **备份**：定期备份数据库
5. **SSL**：使用 HTTPS 加密通信
6. **监控**：设置服务监控和告警

---

## 故障排查

### 服务无法启动

```bash
# 查看详细日志
docker-compose -f docker-compose.prod.yml logs --tail=100

# 检查容器状态
docker-compose -f docker-compose.prod.yml ps

# 检查端口占用
sudo netstat -tulpn | grep 8080
```

### 数据库连接失败

```bash
# 检查数据库容器是否运行
docker-compose -f docker-compose.prod.yml ps db

# 查看数据库日志
docker-compose -f docker-compose.prod.yml logs db

# 测试数据库连接
docker-compose -f docker-compose.prod.yml exec db psql -U blog blog
```

### 更新失败回滚

```bash
# 查看可用镜像版本
docker images ghcr.io/your-username/blog-backend

# 回滚到特定版本
docker-compose -f docker-compose.prod.yml down
# 修改 docker-compose.prod.yml 中的镜像标签
docker-compose -f docker-compose.prod.yml up -d
```

---

## 性能优化

### 1. 数据库连接池

在 `config/db.go` 中配置连接池：

```go
sqlDB, err := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

### 2. 启用 Gzip 压缩

在 Nginx 配置中添加：

```nginx
gzip on;
gzip_types text/plain application/json;
gzip_min_length 1000;
```

### 3. 配置资源限制

在 `docker-compose.prod.yml` 中添加：

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

---

## 多环境部署

可以为不同环境创建不同的配置文件：

- `docker-compose.yml` - 开发环境
- `docker-compose.prod.yml` - 生产环境
- `docker-compose.staging.yml` - 测试环境

使用时指定配置文件：

```bash
docker-compose -f docker-compose.staging.yml up -d
```
