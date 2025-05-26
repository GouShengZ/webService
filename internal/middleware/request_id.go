package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader 请求ID头名称
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey 在gin上下文中存储请求ID的键名
	RequestIDKey = "request_id"
)

// RequestIDMiddleware 请求ID中间件
// 为每个请求生成唯一的ID，用于日志追踪和链路追踪
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头中获取请求ID
		requestID := c.GetHeader(RequestIDHeader)

		// 如果请求头中没有请求ID，则生成一个新的
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 将请求ID存储到gin上下文中
		c.Set(RequestIDKey, requestID)

		// 将请求ID添加到响应头中
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// generateRequestID 生成唯一的请求ID
func generateRequestID() string {
	return uuid.New().String()
}

// GetRequestIDFromContext 从gin上下文中获取请求ID
func GetRequestIDFromContext(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
