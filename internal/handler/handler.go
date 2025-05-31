package handler

import (
	"net/http"
	"strconv"
	"time"

	"webservice/internal/config"
	"webservice/internal/middleware"
	"webservice/internal/minio"
	"webservice/internal/models"
	"webservice/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 处理器结构体
type Handler struct {
	cfg            *config.Config
	db             *gorm.DB
	userService    *service.UserService
	packageService *service.PackageService
	PackageHandler *PackageHandler
}

// NewHandler 创建处理器实例
func NewHandler(cfg *config.Config, db *gorm.DB, minioClient *minio.Client) *Handler {
	userService := service.NewUserService(db)
	packageService := service.NewPackageService(db, minioClient)
	packageHandler := NewPackageHandler(packageService)

	return &Handler{
		cfg:            cfg,
		db:             db,
		userService:    userService,
		packageService: packageService,
		PackageHandler: packageHandler,
	}
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := h.db.DB()
	if err != nil {
		middleware.InternalServerErrorResponse(c, "Database connection error")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		middleware.InternalServerErrorResponse(c, "Database ping failed")
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"database":  "connected",
	})
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationErrorResponse(c, err.Error())
		return
	}

	// 验证用户
	user, err := h.userService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		middleware.UnauthorizedResponse(c, err.Error())
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role, h.cfg.JWT)
	if err != nil {
		middleware.InternalServerErrorResponse(c, "Failed to generate token")
		return
	}

	middleware.SuccessResponse(c, models.LoginResponse{
		User:  user.ToPublicUser(),
		Token: token,
	})
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationErrorResponse(c, err.Error())
		return
	}

	// 创建用户
	user, err := h.userService.CreateUser(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusConflict, err.Error())
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role, h.cfg.JWT)
	if err != nil {
		middleware.InternalServerErrorResponse(c, "Failed to generate token")
		return
	}

	middleware.SuccessResponse(c, models.LoginResponse{
		User:  user.ToPublicUser(),
		Token: token,
	})
}

// RefreshToken 刷新token
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationErrorResponse(c, err.Error())
		return
	}

	// 刷新token
	newToken, err := middleware.RefreshToken(req.Token, h.cfg.JWT)
	if err != nil {
		middleware.UnauthorizedResponse(c, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"token": newToken})
}

// GetProfile 获取用户个人资料
func (h *Handler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		middleware.UnauthorizedResponse(c, "User not found")
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		middleware.NotFoundResponse(c, "User not found")
		return
	}

	middleware.SuccessResponse(c, user.ToPublicUser())
}

// UpdateProfile 更新用户个人资料
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		middleware.UnauthorizedResponse(c, "User not found")
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationErrorResponse(c, err.Error())
		return
	}

	user, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusConflict, err.Error())
		return
	}

	middleware.SuccessResponse(c, user.ToPublicUser())
}

// Logout 用户登出
func (h *Handler) Logout(c *gin.Context) {
	// 在实际应用中，这里可以将token加入黑名单
	// 目前只是返回成功响应
	middleware.SuccessResponse(c, gin.H{"message": "Logged out successfully"})
}

// GetUsers 获取用户列表（管理员）
func (h *Handler) GetUsers(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	role := c.Query("role")
	statusStr := c.Query("status")

	var status models.UserStatus
	if statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status = models.UserStatus(s)
		}
	}

	// 限制分页大小
	if pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	users, total, err := h.userService.GetUsers(page, pageSize, role, status)
	if err != nil {
		middleware.InternalServerErrorResponse(c, "Failed to get users")
		return
	}

	// 转换为公开用户信息
	publicUsers := make([]*models.PublicUser, len(users))
	for i, user := range users {
		publicUsers[i] = user.ToPublicUser()
	}

	middleware.SuccessResponse(c, gin.H{
		"users": publicUsers,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetUser 获取单个用户信息（管理员）
func (h *Handler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.ValidationErrorResponse(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		middleware.NotFoundResponse(c, "User not found")
		return
	}

	middleware.SuccessResponse(c, user.ToPublicUser())
}

// UpdateUser 更新用户信息（管理员）
func (h *Handler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.ValidationErrorResponse(c, "Invalid user ID")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationErrorResponse(c, err.Error())
		return
	}

	user, err := h.userService.UpdateUser(uint(id), &req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusConflict, err.Error())
		return
	}

	middleware.SuccessResponse(c, user.ToPublicUser())
}

// DeleteUser 删除用户（管理员）
func (h *Handler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.ValidationErrorResponse(c, "Invalid user ID")
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		middleware.InternalServerErrorResponse(c, "Failed to delete user")
		return
	}

	middleware.SuccessResponse(c, gin.H{"message": "User deleted successfully"})
}

// GetPublicUsers 获取公开用户列表
func (h *Handler) GetPublicUsers(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 限制分页大小
	if pageSize > 50 {
		pageSize = 50
	}
	if page < 1 {
		page = 1
	}

	users, total, err := h.userService.GetPublicUsers(page, pageSize)
	if err != nil {
		middleware.InternalServerErrorResponse(c, "Failed to get users")
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetPublicUser 获取公开用户信息
func (h *Handler) GetPublicUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		middleware.ValidationErrorResponse(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		middleware.NotFoundResponse(c, "User not found")
		return
	}

	// 只返回活跃用户的公开信息
	if !user.IsActive() {
		middleware.NotFoundResponse(c, "User not found")
		return
	}

	middleware.SuccessResponse(c, user.ToPublicUser())
}
