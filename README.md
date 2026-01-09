# RandImg - 随机图片API服务

一个功能完整的随机图片API服务，支持多图源插件、缓存、限流、代理、格式转换，并提供管理后台。

## 功能特性

- ✅ **随机图片API** - 支持按分类、设备筛选
- ✅ **多种返回格式** - 302重定向、代理模式、JSON格式
- ✅ **图片代理** - 解决跨域问题，支持实时压缩(jpg/png/webp)
- ✅ **API Key认证** - 安全的API访问控制
- ✅ **动态限流** - 基于滑动窗口的限流，可在后台动态调整
- ✅ **使用统计** - 异步批量写入，不影响API性能
- ✅ **管理后台** - 完整的Web UI管理界面
- ✅ **图源插件** - 可扩展的图源插件系统(示例: Unsplash)
- ✅ **Cloudflare友好** - 自动设置Cache-Control头(2天缓存)

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 创建数据目录

```bash
mkdir -p data
```

### 3. 运行服务

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动

### 4. 访问管理后台

打开浏览器访问: `http://localhost:8080/admin`

默认管理员Token: `admin_secret_token` (生产环境请修改)

## API文档

### 公开API

所有公开API需要在Header或Query中提供API Key:
- Header: `X-API-Key: your_api_key`
- Query: `?api_key=your_api_key`

#### 1. 随机图片

```
GET /api/random
```

**参数:**
- `category` (可选) - 分类slug，如 `acg`, `landscape`
- `device` (可选) - 设备类型: `pc`(横屏图片), `mobile`(竖屏图片)
- `format` (可选) - 返回格式: `redirect`(默认), `proxy`, `json`
- `compress` (可选) - 是否压缩: `false`(默认), `true`

**device参数说明:**
- `pc` - 返回横屏图片（宽>高），适合PC端壁纸
- `mobile` - 返回竖屏图片（高>宽），适合手机壁纸
- 不指定 - 返回所有图片（包括正方形）
- 注意：正方形图片（宽=高）不会被pc或mobile筛选返回

**示例:**

```bash
# 302重定向到原图
curl "http://localhost:8080/api/random?api_key=YOUR_KEY"

# 获取PC端横屏图片
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&device=pc"

# 获取移动端竖屏图片
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&device=mobile"

# 代理模式
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&format=proxy"

# JSON格式
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&format=json"

# 指定分类+设备
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&category=acg&device=pc"

# 代理+压缩为WebP
curl "http://localhost:8080/api/random?api_key=YOUR_KEY&format=proxy&compress=true"
```

#### 2. 图片代理

```
GET /api/proxy/:id
```

**参数:**
- `compress` (可选) - 是否压缩: `false`(默认), `true`
- `format` (可选) - 目标格式: `webp`, `jpeg`, `png`

**示例:**

```bash
# 代理图片
curl "http://localhost:8080/api/proxy/1?api_key=YOUR_KEY"

# 压缩为WebP
curl "http://localhost:8080/api/proxy/1?api_key=YOUR_KEY&compress=true&format=webp"
```

### 管理API

所有管理API需要在Header中提供管理员Token:
```
Authorization: Bearer admin_secret_token
```

#### 图片管理

```bash
# 获取图片列表
GET /api/admin/images?page=1&page_size=20&category=acg&status=active

# 获取单个图片
GET /api/admin/images/:id

# 创建图片
POST /api/admin/images
{
  "source_url": "https://example.com/image.jpg",
  "category_id": 1,
  "width": 1920,
  "height": 1080,
  "format": "jpeg",
  "source": "Unsplash"
}

# 更新图片
PUT /api/admin/images/:id
{
  "status": "inactive"
}

# 删除图片(软删除)
DELETE /api/admin/images/:id
```

#### 分类管理

```bash
# 获取分类列表
GET /api/admin/categories

# 创建分类
POST /api/admin/categories
{
  "name": "动漫游戏",
  "slug": "acg",
  "description": "ACG相关图片"
}

# 更新分类
PUT /api/admin/categories/:id

# 删除分类
DELETE /api/admin/categories/:id
```

#### API Key管理

```bash
# 获取API Key列表
GET /api/admin/api-keys

# 创建API Key
POST /api/admin/api-keys
{
  "rate_limit": 60
}

# 更新API Key
PUT /api/admin/api-keys/:id
{
  "rate_limit": 120,
  "status": "active"
}

# 删除API Key
DELETE /api/admin/api-keys/:id
```

#### 统计查询

```bash
# 获取统计概览
GET /api/admin/stats/overview

# 获取详细统计
GET /api/admin/stats?api_key_id=1&start_time=2024-01-01&end_time=2024-01-31
```

## 项目结构

```
randimg/
├── cmd/
│   └── server/
│       └── main.go           # 主程序入口
├── internal/
│   ├── api/
│   │   ├── public.go         # 公开API处理器
│   │   └── admin.go          # 管理API处理器
│   ├── database/
│   │   └── db.go             # 数据库初始化
│   ├── middleware/
│   │   ├── auth.go           # 认证中间件
│   │   └── ratelimit.go      # 限流中间件
│   ├── model/
│   │   └── models.go         # 数据模型
│   ├── plugin/
│   │   └── unsplash.go       # Unsplash插件示例
│   └── service/
│       ├── proxy.go          # 图片代理服务
│       └── stat.go           # 统计服务
├── web/
│   └── dist/
│       ├── index.html        # 管理后台HTML
│       └── app.js            # 管理后台JS
├── data/
│   └── randimg.db            # SQLite数据库
├── go.mod
├── go.sum
└── README.md
```

## 配置说明

### 环境变量

- `PORT` - 服务端口，默认 `8080`

### 管理员Token

默认Token为 `admin_secret_token`，生产环境请修改:

编辑 `internal/middleware/auth.go`:
```go
if token != "your_secure_token_here" {
    // ...
}
```

## 图源插件开发

参考 `internal/plugin/unsplash.go` 实现自定义图源插件:

```go
type YourPlugin struct {
    // ...
}

func (p *YourPlugin) FetchPhotos() ([]Photo, error) {
    // 从图源获取图片
}

func (p *YourPlugin) ImportPhotos(categoryID uint) (int, error) {
    // 导入图片到数据库
}
```

## 部署建议

### 1. 使用Cloudflare CDN

在Cloudflare中配置:
- 缓存规则: 缓存所有 `/api/random` 和 `/api/proxy/*` 请求
- 缓存时间: 2天(已在代码中设置Cache-Control头)

### 2. 使用Systemd

创建 `/etc/systemd/system/randimg.service`:

```ini
[Unit]
Description=RandImg Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/randimg
ExecStart=/opt/randimg/randimg
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

启动服务:
```bash
sudo systemctl enable randimg
sudo systemctl start randimg
```

### 3. 使用Nginx反向代理

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 性能优化

1. **限流**: 根据实际情况调整API Key的限流值
2. **统计批量写入**: 默认50���或10秒批量写入一次
3. **图片压缩**: 仅在需要时启用，避免CPU过载
4. **数据库索引**: 已在关键字段添加索引

## 安全建议

1. **修改管理员Token**: 不要使用默认Token
2. **HTTPS**: 生产环境必须使用HTTPS
3. **API Key保护**: 不要在前端暴露API Key
4. **限流设置**: 合理设置限流值，防止滥用

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request!
