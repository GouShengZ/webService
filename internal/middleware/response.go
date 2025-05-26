package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构体
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ResponseMiddleware 响应格式化中间件
func ResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 如果已经写入了响应，则不再处理
		if c.Writer.Written() {
			return
		}

		// 检查是否有错误
		if len(c.Errors) > 0 {
			// 获取最后一个错误
			err := c.Errors.Last()
			ErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		// 如果没有设置状态码，默认为200
		if c.Writer.Status() == 200 {
			SuccessResponse(c, nil)
		}
	}
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	response := Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: c.GetString("request_id"),
	}
	c.JSON(http.StatusOK, response)
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, httpCode int, message string) {
	response := Response{
		Code:      httpCode,
		Message:   message,
		Timestamp: time.Now().Unix(),
		RequestID: c.GetString("request_id"),
	}
	c.JSON(httpCode, response)
}

// CustomResponse 自定义响应
func CustomResponse(c *gin.Context, httpCode int, code int, message string, data interface{}) {
	response := Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: c.GetString("request_id"),
	}
	c.JSON(httpCode, response)
}

// ValidationErrorResponse 参数验证错误响应
func ValidationErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse 未授权响应
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse 禁止访问响应
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Forbidden"
	}
	ErrorResponse(c, http.StatusForbidden, message)
}

// NotFoundResponse 资源不存在响应
func NotFoundResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Resource not found"
	}
	ErrorResponse(c, http.StatusNotFound, message)
}

// InternalServerErrorResponse 服务器内部错误响应
func InternalServerErrorResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	ErrorResponse(c, http.StatusInternalServerError, message)
}
