// Package main 是 RandImg 的组合根: 只做装配不含业务逻辑.
// 启动配置 (环境变量读取与告警) 属组合根; 模块划分见 .trellis/spec/arch/modules.md.
package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"randimg/internal/access"
	"randimg/internal/channel"
	"randimg/internal/distribution"
	"randimg/internal/ghimport"
	"randimg/internal/imageproxy"
	"randimg/internal/library"
	"randimg/internal/metadata"
	"randimg/internal/metrics"
	"randimg/internal/store"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// version 由构建时 ldflags -X main.version 注入.
var version = "dev"

// healthcheck 自请求 /api/stats, 供容器健康检查调用 (distroless 无 shell 工具).
func healthcheck(port string) int {
	resp, err := http.Get("http://127.0.0.1:" + port + "/api/stats")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func main() {
	check := flag.Bool("healthcheck", false, "self-check /api/stats then exit")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if *check {
		os.Exit(healthcheck(port))
	}

	_ = godotenv.Load()

	log.Printf("RandImg %s starting...", version)

	// ADMIN_TOKEN 未配置时回退默认值并输出告警.
	if os.Getenv("ADMIN_TOKEN") == "" {
		log.Println("[WARN] ADMIN_TOKEN not set, using development default; set it in production")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/randimg.db"
	}
	// data 目录自动创建 (distroless 无 shell, 不能 RUN mkdir).
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}
	if err := store.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 后台服务: 异步计量与元数据补全.
	// 首次迁移: 明细表存量回填按日聚合 (仅聚合表为空时执行).
	if err := store.BackfillDailyCalls(); err != nil {
		log.Printf("[store] backfill daily calls failed: %v", err)
	}
	metrics.Start()
	fetchService := metadata.NewMetadataFetchService(10)
	fetchService.Start()

	// 匿名桶阈值: DB 配置优先, ANON_RATE_LIMIT env 为缺省回退 (默认 300).
	anonLimit := 300
	if v := os.Getenv("ANON_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			anonLimit = n
		}
	}
	access.LoadAnonLimit(anonLimit)

	dist := distribution.NewHandler()
	proxy := imageproxy.NewHandler()
	lib := library.NewHandler(fetchService)
	channels := channel.NewHandler()
	stats := metrics.NewHandler()
	ghimp := ghimport.NewHandler(fetchService)

	r := gin.Default()

	// 公开面 CORS 全局挂载 (预检无匹配路由, 必须在全局链处理).
	r.Use(access.CORSMiddleware())

	// 公开面: Channel 校验与计量 → 限流 (Channel 窗口 + 匿名桶).
	apiGroup := r.Group("/api")
	apiGroup.Use(access.AccessMiddleware())
	apiGroup.Use(access.RateLimitMiddleware())
	{
		apiGroup.GET("/random", dist.RandomImage)
		apiGroup.GET("/proxy/:id", proxy.ProxyImage)
		apiGroup.GET("/images", dist.ListImages)
		apiGroup.GET("/categories", dist.ListCategories)
	}

	// 公开统计: 自身不计量, 受匿名桶兜底限流.
	statsGroup := r.Group("/api")
	statsGroup.Use(access.RateLimitMiddleware())
	statsGroup.GET("/stats", stats.PublicStats)

	// 管理面: Bearer 认证, 无 CORS 头.
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(access.AdminAuthMiddleware())
	{
		adminGroup.GET("/images", lib.ListImages)
		adminGroup.GET("/images/:id", lib.GetImage)
		adminGroup.POST("/images", lib.CreateImage)
		adminGroup.POST("/images/batch", lib.BatchCreateImages)
		adminGroup.PUT("/images/:id", lib.UpdateImage)
		adminGroup.PUT("/images/batch", lib.BatchUpdateImages)
		adminGroup.DELETE("/images/:id", lib.DeleteImage)
		adminGroup.DELETE("/images/batch", lib.BatchDeleteImages)
		adminGroup.POST("/images/auto-fetch", lib.AutoFetchInfo)

		adminGroup.GET("/categories", lib.ListCategories)
		adminGroup.POST("/categories", lib.CreateCategory)
		adminGroup.PUT("/categories/:id", lib.UpdateCategory)
		adminGroup.DELETE("/categories/:id", lib.DeleteCategory)

		adminGroup.GET("/channels", channels.ListChannels)
		adminGroup.POST("/channels", channels.CreateChannel)
		adminGroup.PUT("/channels/:id", channels.UpdateChannel)
		adminGroup.DELETE("/channels/:id", channels.DeleteChannel)

		adminGroup.GET("/stats", stats.AdminStats)
		adminGroup.GET("/stats/overview", stats.OverviewStats)

		adminGroup.GET("/settings/anon", access.AnonSettings)
		adminGroup.PUT("/settings/anon", access.UpdateAnonSettings)

		adminGroup.POST("/import/github", ghimp.ImportFromGitHub)
		adminGroup.POST("/import/github/preview", ghimp.PreviewImport)
	}

	// 静态响应协商缓存: 更新即时生效.
	r.Use(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/" || p == "/admin" || p == "/gallery" || strings.HasPrefix(p, "/assets/") {
			c.Header("Cache-Control", "no-cache")
		}
		c.Next()
	})

	// 三页均为 frontend/ 的 Vue 构建产物.
	r.StaticFile("/admin", "./frontend/dist/admin.html")
	r.Static("/assets", "./frontend/dist/assets")
	r.GET("/", func(c *gin.Context) { c.File("./frontend/dist/home.html") })
	r.GET("/gallery", func(c *gin.Context) { c.File("./frontend/dist/gallery.html") })

	// 优雅关闭.
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		fetchService.Stop()
		metrics.Stop()
		os.Exit(0)
	}()

	log.Printf("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
