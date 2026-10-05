# RandImg

一个轻量级的随机图片 API 服务，支持多图源、智能设备识别、图片代理和管理后台。

## 特性

- 🎲 **随机图片 API** - 按分类、设备类型返回随机图片
- 🔗 **Channel 接入** - 公开接入标识，计量、独立限流与熔断
- 🖼️ **图片代理** - 解决跨域问题，支持压缩和格式转换
- 📱 **智能设备识别** - 自动识别 PC/移动端，返回适配图片
- 🎨 **管理后台** - 完整的 Web UI 管理界面
- 📊 **使用统计** - 异步批量统计，不影响性能
- 🔌 **插件系统** - 可扩展的图源插件

## 快速开始

### Docker 部署（推荐）

```bash
# 下载 compose 文件与环境变量示例
curl -O https://raw.githubusercontent.com/ltba/randimg/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/ltba/randimg/main/.env.example
cp .env.example .env   # 按需修改 ADMIN_TOKEN 等

# 启动服务
docker compose up -d
```

访问 `http://localhost:8080` 查看首页，`http://localhost:8080/admin` 进入管理后台。

镜像标签： `latest` 跟随主分支， `v*` 为语义化版本（tag 发布）， `sha-*` 为具体提交，可按需钉住版本。

### 本地运行

```bash
# 克隆仓库
git clone https://github.com/ltba/randimg.git
cd randimg

# 安装依赖
go mod tidy

# 运行服务
go run cmd/server/main.go
```

## 使用示例

### 获取随机图片

```bash
# 最简单的用法（302 重定向，匿名访问）
curl http://localhost:8080/api/random

# 携带 Channel（计量与独立限流）
curl http://localhost:8080/api/random?channel_id=YOUR_CHANNEL

# 获取 PC 端横屏图片
curl http://localhost:8080/api/random?device=pc

# 获取移动端竖屏图片
curl http://localhost:8080/api/random?device=mobile

# 按分类获取
curl http://localhost:8080/api/random?category=acg

# JSON 格式
curl http://localhost:8080/api/random?output=json
```

### HTML 中使用

```html
<!-- 直接作为图片源 -->
<img src="http://localhost:8080/api/random?channel_id=YOUR_CHANNEL" />

<!-- 指定分类和设备 -->
<img src="http://localhost:8080/api/random?channel_id=YOUR_CHANNEL&category=acg&device=pc" />
```

## 管理后台

访问 `http://localhost:8080/admin` 进入管理后台。

默认管理员 Token：`admin_secret_token`（生产环境请修改 `.env` 文件中的 `ADMIN_TOKEN`）

在管理后台可以：
- 管理图片和分类
- 创建和管理 Channel（限流与来源绑定）
- 查看使用统计
- 使用脚本工具批量导入图片

## 环境变量

创建 `.env` 文件：

```env
PORT=8080
DB_PATH=data/randimg.db
ADMIN_TOKEN=your_secure_token_here
ANON_RATE_LIMIT=300
```

## 技术栈

- **后端**: Go 1.23 + Gin + GORM + SQLite
- **前端**: 原生 HTML/CSS/JavaScript
- **部署**: Docker + GitHub Actions

## 许可证

GPL V3

