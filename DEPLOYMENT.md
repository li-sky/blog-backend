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

```bash
# 执行部署脚本
chmod +x deploy.sh
./deploy.sh
```

### 3. 验证部署

```bash
# 查看服务状态
docker-compose -f docker-compose.prod.yml ps

# 查看日志
docker-compose -f docker-compose.prod.yml logs -f api

# 测试 API
curl http://localhost:8080/health
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

    client_max_body_size 10M;

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
