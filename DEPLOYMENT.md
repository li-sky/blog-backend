# 部署指南

## 架构

图片处理发生在管理端浏览器中。编辑器使用 Web Worker + WebAssembly
生成三份固定 WebP，然后与原图一起上传：

```text
浏览器
  ├─ 原图
  ├─ <uuid>_w600.webp
  ├─ <uuid>_w1200.webp
  └─ <uuid>_full.webp
          │
          ▼
      Go API ─── PostgreSQL
          │
          ▼
      /uploads
          │
          ▼
   Nginx 静态文件服务
```

动画 GIF 不生成静态变体。历史图片由管理后台的 `Upgrade Images`
页面下载、重编码并上传。

## 环境变量

```env
DATABASE_URL=postgres://blog:password@db:5432/blog?sslmode=disable
JWT_SECRET=replace-with-a-long-random-secret
GIN_MODE=release

STORAGE_BACKEND=filesystem
STORAGE_FS_BASE_PATH=/uploads
STORAGE_FS_BASE_URL=https://image.yourdomain.com
```

`STORAGE_FS_BASE_URL` 必须指向 Nginx 暴露同一上传目录的公开地址。
如果图片域名与前端域名不同，必须保留静态路由上的 CORS 响应头，
否则历史图片升级页面无法读取原图。

## Docker Compose

开发环境：

```bash
docker compose up -d --build
```

服务包括 PostgreSQL、Go API 和 Nginx。上传目录作为共享卷挂载到
API（读写）与 Nginx（只读）。

生产环境：

```bash
docker compose -f docker-compose.prod.yml up -d
```

生产 Compose 保留 API、数据库和备份服务。可使用宿主机 Nginx 或
单独的 Nginx 容器暴露 `/data/uploads`。

## Nginx

同域名开发配置：

```nginx
location /images/ {
    alias /uploads/;
    add_header Cache-Control "public, max-age=31536000, immutable";
    add_header Access-Control-Allow-Origin "*" always;
    add_header X-Content-Type-Options "nosniff";
}
```

独立生产图片域名：

```nginx
server {
    listen 443 ssl http2;
    server_name image.yourdomain.com;

    location / {
        alias /data/uploads/;
        add_header Cache-Control "public, max-age=31536000, immutable";
        add_header Access-Control-Allow-Origin "*" always;
        add_header X-Content-Type-Options "nosniff";
    }
}
```

API 上传限制应覆盖原图和三个 WebP 的总请求大小：

```nginx
client_max_body_size 160M;

location /api/v1/images/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_request_buffering off;
    proxy_buffering off;
}
```

## 数据库升级

应用启动时 `AutoMigrate` 会为 `images` 表添加 WebP variant URL 和
文件名字段。升级应用后进入管理后台：

1. 打开 **Upgrade Images**。
2. 点击 **开始升级**。
3. 保持页面打开，等待所有图片变为“已完成”或“无需转换”。
4. 对失败项使用“重试失败项”。

页面使用四个并发任务处理图片。转换接口会验证 WebP 格式、目标尺寸
与原图长宽比。

## 验证

```bash
curl https://api.yourdomain.com/health
curl -I https://image.yourdomain.com/<uuid>.jpg
curl -I https://image.yourdomain.com/<uuid>_w600.webp
curl -I https://image.yourdomain.com/<uuid>_w1200.webp
curl -I https://image.yourdomain.com/<uuid>_full.webp
```

确认静态图片响应包含长期缓存头和 `Access-Control-Allow-Origin`。

## 备份与回滚

必须同时备份 PostgreSQL 和整个上传目录。原图始终保留，因此可以
在未来重新生成其他规格。回滚前端不会破坏原图；数据库中新添加的
variant 字段也可由旧版本忽略。
