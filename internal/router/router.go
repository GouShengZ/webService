package router

import (
	"net/http"

	"webservice/internal/config"
	"webservice/internal/handler"
	"webservice/internal/middleware"
	"webservice/internal/minio"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(cfg *config.Config, db *gorm.DB, minioClient *minio.Client) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建Gin引擎
	r := gin.New()

	// 设置信任的代理
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// 全局中间件
	setupMiddleware(r, cfg)

	// 设置路由组
	setupRoutes(r, cfg, db, minioClient)

	return r
}

// setupMiddleware 设置全局中间件
func setupMiddleware(r *gin.Engine, cfg *config.Config) {
	// 恢复中间件（处理panic）
	r.Use(gin.Recovery())

	// 请求ID中间件
	r.Use(middleware.RequestIDMiddleware())

	// 链路追踪中间件
	r.Use(middleware.TracingMiddleware())

	// 日志中间件
	r.Use(middleware.LoggerMiddleware())

	// CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
	}))

	// 响应格式化中间件
	r.Use(middleware.ResponseMiddleware())
}

// setupRoutes 设置路由组
func setupRoutes(r *gin.Engine, cfg *config.Config, db *gorm.DB, minioClient *minio.Client) {
	// 创建处理器
	h := handler.NewHandler(cfg, db, minioClient)

	// 健康检查路由 - 用于监控服务状态
	r.GET("/health", h.HealthCheck)       // 返回服务健康状态信息
	r.GET("/ping", func(c *gin.Context) { // 简单的连通性测试接口
		middleware.SuccessResponse(c, gin.H{"message": "pong"})
	})

	// API版本1路由组 - 所有业务API的根路径
	v1 := r.Group("/api/v1")
	{
		// 公开路由（不需要认证）- 任何人都可以访问的接口
		public := v1.Group("/public")
		{
			public.POST("/login", h.Login)          // 用户登录接口 - 验证用户名密码并返回JWT token
			public.POST("/register", h.Register)    // 用户注册接口 - 创建新用户账户
			public.POST("/refresh", h.RefreshToken) // Token刷新接口 - 在token即将过期时获取新token
		}

		// 需要认证的路由 - 必须携带有效JWT token才能访问
		auth := v1.Group("/auth")
		// auth.Use(middleware.JWTAuth(cfg.JWT)) // 应用JWT认证中间件
		{
			auth.GET("/profile", h.GetProfile)    // 获取当前用户个人资料
			auth.PUT("/profile", h.UpdateProfile) // 更新当前用户个人资料
			auth.POST("/logout", h.Logout)        // 用户登出接口
		}

		// 管理员路由 - 只有管理员角色才能访问的接口
		admin := v1.Group("/admin")
		// admin.Use(middleware.JWTAuth(cfg.JWT))  // 应用JWT认证中间件
		// admin.Use(middleware.RoleAuth("admin")) // 应用角色权限中间件，限制只有admin角色可访问
		{
			admin.GET("/users", h.GetUsers)          // 获取用户列表 - 支持分页和筛选
			admin.GET("/users/:id", h.GetUser)       // 根据ID获取指定用户详细信息
			admin.PUT("/users/:id", h.UpdateUser)    // 更新指定用户信息
			admin.DELETE("/users/:id", h.DeleteUser) // 删除指定用户（软删除）
		}

		// 用户路由 - 公开的用户信息查询接口
		users := v1.Group("/users")
		// users.Use(middleware.OptionalJWTAuth(cfg.JWT)) // 可选认证中间件，有token时解析用户信息，无token时也允许访问
		{
			users.GET("/", h.GetPublicUsers)   // 获取公开用户列表 - 只返回公开信息
			users.GET("/:id", h.GetPublicUser) // 根据ID获取指定用户的公开信息
		}

		// 包管理路由 - 包的创建、更新、删除等操作
		packages := v1.Group("/packages")
		{
			// 公开的包相关接口（不需要认证）
			packages.GET("/", h.PackageHandler.SearchPackages)                      // 搜索包列表 - 支持关键词、作者等筛选
			packages.GET("/stats", h.PackageHandler.GetPackageStats)                // 获取包统计信息 - 总数、下载量等
			packages.GET("/:package", h.PackageHandler.GetPackage)                  // 获取指定包的详细信息
			packages.GET("/:package/versions", h.PackageHandler.GetPackageVersions) // 获取指定包的所有版本列表

			// 包版本下载接口（支持匿名下载公开包）
			packages.GET("/:package/:version/download", h.PackageHandler.DownloadPackageVersion) // 直接下载包文件
			packages.GET("/:package/:version/download-url", h.PackageHandler.GetDownloadURL)     // 获取下载链接

			// 需要认证的包管理接口
			packagesAuth := packages.Group("/update")
			// packagesAuth.Use(middleware.JWTAuth(cfg.JWT))
			{
				packagesAuth.POST("/", h.PackageHandler.CreatePackage)                           // 创建新包
				packagesAuth.PUT("/:package", h.PackageHandler.UpdatePackage)                    // 更新包信息
				packagesAuth.DELETE("/:package", h.PackageHandler.DeletePackage)                 // 删除包
				packagesAuth.POST("/:package/versions", h.PackageHandler.UploadPackageVersion)   // 上传新版本
				packagesAuth.DELETE("/:package/:version", h.PackageHandler.DeletePackageVersion) // 删除指定版本
			}
		}
	}

	// 404处理 - 当请求的路由不存在时返回404错误
	r.NoRoute(func(c *gin.Context) {
		middleware.NotFoundResponse(c, "Route not found")
	})

	// 405处理 - 当请求方法不被允许时返回405错误
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"code":      http.StatusMethodNotAllowed,
			"message":   "Method not allowed",
			"timestamp": middleware.GetRequestIDFromContext(c),
		})
	})
}
