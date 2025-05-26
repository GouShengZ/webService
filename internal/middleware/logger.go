package middleware

import (
	"bytes"
	"io"
	"time"

	"webservice/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// responseWriter 自定义响应写入器，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 重写Write方法以捕获响应内容
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		startTime := time.Now()

		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建自定义响应写入器
		responseWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseWriter

		// 处理请求
		c.Next()

		// 计算处理时间
		latency := time.Since(startTime)

		// 获取请求信息
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()
		referer := c.Request.Referer()

		// 构建完整路径
		if raw != "" {
			path = path + "?" + raw
		}

		// 构建日志字段
		fields := logrus.Fields{
			"client_ip":     clientIP,
			"method":        method,
			"path":          path,
			"status_code":   statusCode,
			"latency":       latency.String(),
			"latency_ms":    latency.Milliseconds(),
			"user_agent":    userAgent,
			"referer":       referer,
			"request_size":  len(requestBody),
			"response_size": responseWriter.body.Len(),
		}

		// 添加请求ID（如果存在）
		if requestID := c.GetString("request_id"); requestID != "" {
			fields["request_id"] = requestID
		}

		// 添加用户ID（如果存在）
		if userID := c.GetString("user_id"); userID != "" {
			fields["user_id"] = userID
		}

		// 根据状态码选择日志级别
		logEntry := logger.WithFields(fields)
		switch {
		case statusCode >= 500:
			logEntry.Error("Server error")
		case statusCode >= 400:
			logEntry.Warn("Client error")
		case statusCode >= 300:
			logEntry.Info("Redirection")
		default:
			logEntry.Info("Request completed")
		}
	}
}
