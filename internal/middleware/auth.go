package middleware

import (
	"errors"
	"strings"
	"time"

	"webservice/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT声明结构体
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth JWT认证中间件
func JWTAuth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		token := getTokenFromHeader(c)
		if token == "" {
			UnauthorizedResponse(c, "Missing authorization token")
			c.Abort()
			return
		}

		// 解析和验证token
		claims, err := parseToken(token, cfg.Secret)
		if err != nil {
			UnauthorizedResponse(c, "Invalid token: "+err.Error())
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// OptionalJWTAuth 可选的JWT认证中间件（不强制要求token）
func OptionalJWTAuth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		token := getTokenFromHeader(c)
		if token != "" {
			// 如果有token，尝试解析
			claims, err := parseToken(token, cfg.Secret)
			if err == nil {
				// 解析成功，将用户信息存储到上下文中
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("role", claims.Role)
			}
		}

		c.Next()
	}
}

// RoleAuth 角色权限中间件
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		userRole, exists := c.Get("role")
		if !exists {
			ForbiddenResponse(c, "User role not found")
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			ForbiddenResponse(c, "Invalid user role")
			c.Abort()
			return
		}

		// 检查角色权限
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		ForbiddenResponse(c, "Insufficient permissions")
		c.Abort()
	}
}

// getTokenFromHeader 从请求头中获取token
func getTokenFromHeader(c *gin.Context) string {
	// 从Authorization头获取
	auth := c.GetHeader("Authorization")
	if auth != "" {
		// Bearer token格式
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	// 从X-Token头获取
	return c.GetHeader("X-Token")
}

// parseToken 解析JWT token
func parseToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateToken 生成JWT token
func GenerateToken(userID uint, username, role string, cfg config.JWTConfig) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.ExpireTime)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// RefreshToken 刷新JWT token
func RefreshToken(tokenString string, cfg config.JWTConfig) (string, error) {
	claims, err := parseToken(tokenString, cfg.Secret)
	if err != nil {
		return "", err
	}

	// 检查token是否即将过期（在过期前30分钟内可以刷新）
	if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
		return "", errors.New("token is not eligible for refresh")
	}

	// 生成新token
	return GenerateToken(claims.UserID, claims.Username, claims.Role, cfg)
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GetUsernameFromContext 从上下文中获取用户名
func GetUsernameFromContext(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	name, ok := username.(string)
	return name, ok
}

// GetRoleFromContext 从上下文中获取用户角色
func GetRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
