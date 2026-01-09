package main

import (
	"log"
	"os"
	"os/signal"
	"randimg/internal/api"
	"randimg/internal/database"
	"randimg/internal/middleware"
	"randimg/internal/service"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// 初始化数据库
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/randimg.db"
	}
	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 创建Gin引擎
	r := gin.Default()

	// 智能CORS中间件：有API key才返回跨域头
	r.Use(func(c *gin.Context) {
		// 检查是否有API key或admin token
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}
		adminToken := c.GetHeader("Authorization")

		// 如果有认证信息，返回CORS头允许跨域
		if apiKey != "" || adminToken != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 创建API处理器
	publicAPI := api.NewPublicAPI()
	adminAPI := api.NewAdminAPI()

	// 公开API路由（需要API key认证和限流）
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware())
	apiGroup.Use(middleware.RateLimitMiddleware())
	{
		apiGroup.GET("/random", publicAPI.RandomImage)
		apiGroup.GET("/proxy/:id", publicAPI.ProxyImage)
		apiGroup.GET("/images", publicAPI.ListImages)
		apiGroup.GET("/categories", publicAPI.ListCategories)
	}

	// 公开统计API（无需认证）
	r.GET("/api/stats", publicAPI.GetPublicStats)

	// 管理API路由（需要管理员认证）
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.AdminAuthMiddleware())
	{
		// 图片管理
		adminGroup.GET("/images", adminAPI.ListImages)
		adminGroup.GET("/images/:id", adminAPI.GetImage)
		adminGroup.POST("/images", adminAPI.CreateImage)
		adminGroup.POST("/images/batch", adminAPI.BatchCreateImages)
		adminGroup.PUT("/images/:id", adminAPI.UpdateImage)
		adminGroup.DELETE("/images/:id", adminAPI.DeleteImage)
		adminGroup.POST("/images/auto-fetch", adminAPI.AutoFetchImageInfo)
		adminGroup.PUT("/images/batch", adminAPI.BatchUpdateImages)
		adminGroup.DELETE("/images/batch", adminAPI.BatchDeleteImages)

		// 分类管理
		adminGroup.GET("/categories", adminAPI.ListCategories)
		adminGroup.POST("/categories", adminAPI.CreateCategory)
		adminGroup.PUT("/categories/:id", adminAPI.UpdateCategory)
		adminGroup.DELETE("/categories/:id", adminAPI.DeleteCategory)

		// API Key管理
		adminGroup.GET("/api-keys", adminAPI.ListAPIKeys)
		adminGroup.POST("/api-keys", adminAPI.CreateAPIKey)
		adminGroup.PUT("/api-keys/:id", adminAPI.UpdateAPIKey)
		adminGroup.DELETE("/api-keys/:id", adminAPI.DeleteAPIKey)

		// 统计查询
		adminGroup.GET("/stats", adminAPI.GetStats)
		adminGroup.GET("/stats/overview", adminAPI.GetStatsOverview)
	}

	// 静态文件服务（管理后台）
	r.Static("/admin", "./web/dist")

	// 静态资源（CSS/JS）
	r.Static("/css", "./web/dist/css")
	r.Static("/js", "./web/dist/js")

	// 首页
	r.GET("/", func(c *gin.Context) {
		c.File("./web/dist/home.html")
	})

	// Gallery页面
	r.GET("/gallery", func(c *gin.Context) {
		c.File("./web/dist/gallery.html")
	})

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		service.GetStatService().Stop()
		service.GetImageFetchService().Stop()
		os.Exit(0)
	}()

	// 启动后台服务
	log.Println("Starting background services...")
	service.GetImageFetchService() // 启动fetch服务

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
