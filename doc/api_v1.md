# 博客后端 API v1 接口文档

> 基于 Gin 的 RESTful API，所有响应均为 JSON。

## 通用说明

- 基础地址：`/api/v1`
- 认证方式：HTTP Header 中携带 `Authorization: Bearer <token>`（标记为「需要登录」的接口必须携带）
- 响应格式：
  - 成功：`200/201/204`，body 为对象或列表
  - 失败：`4xx/5xx`，body 一般为：`{"error": "错误信息"}`
- 分页参数（如支持）：
  - `limit`：每页数量，默认 20
  - `offset`：偏移量，默认 0

---

## 1. 认证模块 Auth

### 1.1 注册普通用户
- 方法：`POST`
- 路径：`/api/v1/auth/register`
- 是否登录：否
- 请求体：
```json
{
  "username": "string",
  "email": "user@example.com",
  "password": "长度>=12的字符串"
}
```
- 响应体：
```json
{
  "token": "jwt-token",
  "user": {
    "id": 1,
    "username": "string",
    "email": "user@example.com",
    "roles": ["role1", "role2"],
    "createdAt": "2025-01-01T00:00:00Z"
  }
}
```

### 1.2 登录
- 方法：`POST`
- 路径：`/api/v1/auth/login`
- 是否登录：否
- 请求体：
```json
{
  "username": "string",
  "password": "string"
}
```
- 响应体：同「注册普通用户」。

### 1.3 首次初始化管理员
- 方法：`POST`
- 路径：`/api/v1/auth/admin/init`
- 是否登录：否（仅用于系统首次初始化）
- 请求体：同「注册普通用户」，创建时会自动赋予 `admin` 角色。
- 响应体：同「注册普通用户」。

### 1.4 获取当前登录用户信息
- 方法：`GET`
- 路径：`/api/v1/auth/me`
- 是否登录：是
- Header：`Authorization: Bearer <token>`
- 响应体：`userDTO`：
```json
{
  "id": 1,
  "username": "string",
  "email": "user@example.com",
  "roles": ["admin"],
  "createdAt": "2025-01-01T00:00:00Z"
}
```

---

## 2. 用户模块 Users

> 所有用户管理接口均需要登录，一般要求管理员或具备相应权限的角色，具体权限由后端服务控制，前端只需带上 token。

### 2.1 用户列表
- 方法：`GET`
- 路径：`/api/v1/users`
- 是否登录：是
- 查询参数：
  - `limit`（可选）
  - `offset`（可选）
- 响应体：
```json
{
  "items": [
    {
      "id": 1,
      "username": "string",
      "email": "user@example.com",
      "roles": ["admin"],
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 2.2 获取单个用户详情
- 方法：`GET`
- 路径：`/api/v1/users/:id`
- 是否登录：是
- 路径参数：`id` 用户 ID
- 响应体：同 `userDTO`。

### 2.3 更新用户信息
- 方法：`PUT`
- 路径：`/api/v1/users/:id`
- 是否登录：是
- 请求体（字段均可选）：
```json
{
  "username": "string",
  "email": "user@example.com",  // 若传则需满足邮箱格式
  "password": "string"
}
```
- 响应体：更新后的 `userDTO`。

### 2.4 删除用户
- 方法：`DELETE`
- 路径：`/api/v1/users/:id`
- 是否登录：是
- 响应：`204 No Content`。

### 2.5 设置用户角色
- 方法：`PUT`
- 路径：`/api/v1/users/:id/roles`
- 是否登录：是
- 请求体：
```json
{
  "roleIds": [1, 2, 3]
}
```
- 响应体：更新后的 `userDTO`。

### 2.6 创建管理员用户
- 方法：`POST`
- 路径：`/api/v1/users/admin`
- 是否登录：是（通常要求当前操作者本身为管理员）
- 请求体：同注册：
```json
{
  "username": "string",
  "email": "admin@example.com",
  "password": "长度>=12的字符串"
}
```
- 响应体：新建管理员的 `userDTO`。

---

## 3. 角色与权限模块 Roles

> 所有接口需要登录，一般需管理员权限。

### 3.1 角色列表
- 方法：`GET`
- 路径：`/api/v1/roles`
- 是否登录：是
- 查询参数：`limit, offset`
- 响应体：
```json
{
  "items": [
    {
      "id": 1,
      "name": "admin",
      "permissions": [
        {"name": "post.manage", "description": "管理文章"}
      ]
    }
  ],
  "total": 1
}
```

### 3.2 获取角色详情
- 方法：`GET`
- 路径：`/api/v1/roles/:id`
- 是否登录：是
- 响应体：单个角色对象，同上结构。

### 3.3 创建角色
- 方法：`POST`
- 路径：`/api/v1/roles`
- 是否登录：是
- 请求体：
```json
{
  "name": "editor",
  "permissions": [
    {
      "name": "post.manage",
      "description": "管理文章"
    }
  ]
}
```
- 响应体：新建角色对象。

### 3.4 更新角色
- 方法：`PUT`
- 路径：`/api/v1/roles/:id`
- 是否登录：是
- 请求体（字段可选）：
```json
{
  "name": "editor",
  "permissions": [
    {
      "name": "post.manage",
      "description": "管理文章"
    }
  ]
}
```
- 响应体：更新后的角色对象。

### 3.5 删除角色
- 方法：`DELETE`
- 路径：`/api/v1/roles/:id`
- 是否登录：是
- 响应：`204 No Content`。

---

## 4. 文章模块 Posts

### 4.1 获取文章列表（已发布）
- 方法：`GET`
- 路径：`/api/v1/posts`
- 是否登录：否
- 查询参数：
  - `limit`（可选）
  - `offset`（可选）
  - `sort`（可选，默认 `created_at`）
  - `order`（可选，`desc`/`asc`，默认 `desc`）
- 响应示例：
```json
{
  "items": [
    {
      "id": 1,
      "title": "文章标题",
      "summary": "文章摘要",
      "userId": 2,
      "status": "published",
      "publishedAt": "2025-01-01T00:00:00Z",
      "createdAt": "2025-01-01T00:00:00Z",
      "updatedAt": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```
- 说明：列表 DTO 不再包含 `body`（正文），如需正文请调用「获取文章详情」。

### 4.2 获取文章详情
- 方法：`GET`
- 路径：`/api/v1/posts/:postID`
- 是否登录：否（若登录且有 `edit_post` 权限或为文章作者，可获取任意状态文章；否则仅能获取已发布文章）
- 路径参数：`postID` 文章 ID
- 响应体：单篇文章完整对象（含 `body`）。
- 说明：`user` 字段不会返回明文邮箱，仅提供 `emailSha256`（对邮箱 `trim + lower` 后计算 SHA-256，再输出 hex）。

### 4.3 创建文章（POST /posts）
- 方法：`POST`
- 路径：`/api/v1/posts`
- 是否登录：是
- 请求体：JSON 将被直接绑定到 `models.Post`，常用字段示例：
```json
{
  "title": "文章标题",
  "summary": "摘要",
  "body": "markdown 或 HTML 正文",
  "status": "draft"
}
```
- 说明：`userId`、`ipAddress` 等由后端根据当前登录用户和请求信息自动填充。
- 响应体：`201 Created`，body 为创建后的文章完整对象。

### 4.4 更新文章（PUT /posts/:postID）
- 方法：`PUT`
- 路径：`/api/v1/posts/:postID`
- 是否登录：是
- 路径参数：`postID` 文章 ID
- 请求体：与创建文章相同结构，例如：
```json
{
  "title": "文章标题（更新）",
  "summary": "更新后的摘要",
  "body": "更新后的正文",
  "status": "draft"
}
```
- 说明：控制器内部会使用路径参数 `postID` 作为文章 ID，忽略 body 中的 `id` 字段。
- 响应体：`200 OK`，返回更新后的文章对象。

### 4.5 发布文章
- 方法：`POST`
- 路径：`/api/v1/posts/:postID/publish`
- 是否登录：是
- 响应：`204 No Content`。

### 4.6 删除文章
- 方法：`DELETE`
- 路径：`/api/v1/posts/:postID`
- 是否登录：是
- 响应：`204 No Content`。

### 4.7 管理文章列表（包含草稿等）
- 方法：`GET`
- 路径：`/api/v1/posts/manage`
- 是否登录：是
- 查询参数：同「获取文章列表」。
- 响应体：返回所有状态的文章列表，结构与「获取文章列表」一致。

---

## 5. 评论模块 Comments

### 5.1 获取文章评论列表
- 方法：`GET`
- 路径：`/api/v1/posts/:postID/comments`
- 是否登录：否
- 查询参数：`limit, offset`
- 响应体：
```json
{
  "items": [
    {
      "id": 1,
      "postId": 1,
      "userId": 2,
      "body": "评论内容",
      "visible": true,
      "user": {
        "id": 2,
        "username": "alice",
        "emailSha256": "<sha256-hex>"
      },
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

- 说明：评论返回的 `user` 不包含明文 `email`；如需做匿名识别，可使用 `emailSha256`（对 email 进行 `trim + lower` 规范化后计算 SHA-256，再以 hex 输出）。

### 5.2 发表评论
- 方法：`POST`
- 路径：`/api/v1/posts/:postID/comments`
- 是否登录：是
- 请求体：
```json
{
  "body": "评论内容"
}
```
- 响应体：创建后的评论对象（不会返回 `user.email`；如包含 `user`，仅提供 `emailSha256`）。

### 5.3 删除评论
- 方法：`DELETE`
- 路径：`/api/v1/posts/:postID/comments/:commentID`
- 是否登录：是
- 响应：`204 No Content`。

### 5.4 设置评论可见性
- 方法：`PUT`
- 路径：`/api/v1/posts/:postID/comments/:commentID/visibility`
- 是否登录：是
- 请求体：
```json
{
  "visible": true
}
```
- 响应：`200 OK`，无 body 或简单 `{}`。

---

## 6. 设置模块 Settings

### 6.1 获取所有设置
- 方法：`GET`
- 路径：`/api/v1/settings`
- 是否登录：否
- 响应体：
```json
{
  "items": [
    {
      "key": "site_title",
      "value": "我的博客",
      "description": "站点标题"
    }
  ]
}
```

### 6.2 更新某个设置
- 方法：`PUT`
- 路径：`/api/v1/settings/:key`
- 是否登录：是（通常为管理员）
- 请求体：
```json
{
  "value": "新的值",
  "description": "可选说明"
}
```
- 响应体：更新后的设置对象。

---

## 7. 图床模块 Images

**权限说明**：
- `upload_image` - 上传图片权限（管理员角色默认拥有）
- `edit_any_image` - 编辑任意用户图片权限（管理员角色）
- `delete_any_image` - 删除任意用户图片权限（管理员角色）
- 普通用户只能编辑和删除自己上传的图片

### 7.1 上传图片
- 方法：`POST`
- 路径：`/api/v1/images/upload`
- 是否登录：是
- 所需权限：`upload_image`
- 请求格式：`multipart/form-data`
- 表单字段：
  - `file`：图片文件（必需，支持 jpg, jpeg, png, gif, webp 格式）
  - `alt`：图片描述文本（可选）
- 响应体：
```json
{
  "id": 1,
  "filename": "abc123.jpg",
  "originalName": "photo.jpg",
  "alt": "我的照片",
  "mimeType": "image/jpeg",
  "size": 102400,
  "width": 1920,
  "height": 1080,
  "url": "https://image.yourdomain.com/abc123.jpg",
  "userId": 1,
  "storageBackend": "filesystem",
  "storagePath": "uploads/abc123.jpg",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

### 7.2 获取图片列表
- 方法：`GET`
- 路径：`/api/v1/images`
- 是否登录：是
- 查询参数：
  - `limit`（可选，默认 20）
  - `offset`（可选，默认 0）
- 响应体：
```json
{
  "items": [
    {
      "id": 1,
      "filename": "abc123.jpg",
      "originalName": "photo.jpg",
      "alt": "我的照片",
      "mimeType": "image/jpeg",
      "size": 102400,
      "width": 1920,
      "height": 1080,
      "url": "https://image.yourdomain.com/abc123.jpg",
      "webp600Url": "https://image.yourdomain.com/abc123_w600.webp",
      "webp1200Url": "https://image.yourdomain.com/abc123_w1200.webp",
      "variantsComplete": true,
      "userId": 1,
      "storageBackend": "filesystem",
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### 7.3 获取图片详情
- 方法：`GET`
- 路径：`/api/v1/images/:id`
- 是否登录：是
- 响应体：单个图片对象，同上结构。

### 7.4 更新图片信息
- 方法：`PUT`
- 路径：`/api/v1/images/:id`
- 是否登录：是
- 权限要求：必须是图片所有者，或拥有 `edit_any_image` 权限
- 请求体：
```json
{
  "alt": "更新后的图片描述"
}
```
- 响应体：更新后的图片对象。

### 7.5 上传历史图片变体
- 方法：`PUT`
- 路径：`/api/v1/images/:id/variants`
- 是否登录：是
- 权限要求：必须是图片所有者，或拥有 `edit_any_image` 权限
- 请求格式：`multipart/form-data`
  - `webp600`：宽度为 `min(原图宽度, 600)` 的 WebP
  - `webp1200`：宽度为 `min(原图宽度, 1200)` 的 WebP
- 服务端会验证格式、尺寸和长宽比。

### 7.6 删除图片
- 方法：`DELETE`
- 路径：`/api/v1/images/:id`
- 是否登录：是
- 权限要求：必须是图片所有者，或拥有 `delete_any_image` 权限
- 说明：同时删除数据库记录和存储后端的文件。
- 响应：`204 No Content`。

**存储后端配置说明**：
- 当前支持的存储后端：`filesystem`（文件系统）
- 使用环境变量 `STORAGE_BACKEND` 配置存储类型（默认 `filesystem`）
- 文件系统存储配置：
  - `STORAGE_FS_BASE_PATH`：文件存储基础路径（默认 `./uploads`）
  - `STORAGE_FS_BASE_URL`：文件访问完整 URL 前缀（如 `https://image.yourdomain.com` 或 `https://cdn.example.com/images`）
- 上传限制：
  - 单个文件最大 50MB
  - 支持的图片格式：jpg, jpeg, png, gif, webp
  - 文件名自动生成（使用 UUID + 原扩展名），避免冲突
- URL 生成规则：返回的 `url` 字段 = `STORAGE_FS_BASE_URL` + `/` + `filename`
- 迁移友好：修改 `STORAGE_FS_BASE_URL` 即可切换到 CDN 或对象存储域名，无需修改代码

**静态图片变体**：

图片上传时由前端生成固定的 WebP 文件，Nginx 直接从上传目录提供文件：

- 原图：`https://image.yourdomain.com/abc123.jpg`
- 600px：`https://image.yourdomain.com/abc123_w600.webp`
- 1200px：`https://image.yourdomain.com/abc123_w1200.webp`
- **响应式图片示例**：
```html
<picture>
  <source 
    srcset="https://image.yourdomain.com/abc123_w600.webp"
    type="image/webp" 
    media="(max-width: 768px)">
  <source 
    srcset="https://image.yourdomain.com/abc123_w1200.webp"
    type="image/webp" 
    media="(min-width: 769px)">
  <img 
    src="https://image.yourdomain.com/abc123.jpg"
    alt="responsive image" 
    loading="lazy">
</picture>
```

**优势**：
- ✅ 阅读请求不触发图片编码
- ✅ 文件 URL 不包含动态转换参数
- ✅ 可以使用长期 immutable 缓存
- ✅ 原图始终保留，未来可重新生成其他规格

---

## 8. RSS/Atom Feed 订阅

### 8.1 Atom Feed
- 方法：`GET`
- 路径：`/feed/atom`
- 是否登录：否
- 响应格式：`application/atom+xml`
- 说明：返回最新 20 篇已发布文章的 Atom 1.0 格式 feed，可用于 RSS 阅读器订阅。

### 8.2 RSS Feed
- 方法：`GET`
- 路径：`/feed/rss`
- 是否登录：否
- 响应格式：`application/rss+xml`
- 说明：返回最新 20 篇已发布文章的 RSS 2.0 格式 feed，可用于 RSS 阅读器订阅。

**Feed 配置说明**：
- Feed 标题、站点 URL、描述等信息从 `settings` 表中读取：
  - `site_title`：站点标题
  - `site_url`：站点 URL（如 `https://example.com`）
  - `site_description`：站点描述
- 如果未配置这些设置，会使用默认值或从请求中推断。

---

## 9. 错误码与前端处理建议

- `400 Bad Request`：参数校验失败或业务校验失败，前端可根据 `error` 字段提示具体原因。
- `401 Unauthorized`：未登录或 token 失效，前端应跳转登录页或重新发起登录。
- `403 Forbidden`：无权限访问，前端可提示「无权限」并禁止当前操作。
- `404 Not Found`：资源不存在，例如文章/评论已被删除。
- `409 Conflict`：冲突错误，例如注册时用户名或邮箱已存在。
- `500 Internal Server Error`：服务器异常，前端统一提示「服务器开小差了，请稍后重试」。

前端需要统一封装请求拦截器，在有 token 时自动附加 `Authorization` 头，并对上述状态码进行集中处理。
