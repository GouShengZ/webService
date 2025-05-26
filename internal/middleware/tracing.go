package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// TracingMiddleware 链路追踪中间件
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取全局tracer
		tracer := opentracing.GlobalTracer()

		// 尝试从请求头中提取span上下文
		spanCtx, _ := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.Request.Header),
		)

		// 创建操作名称
		operationName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if operationName == " " {
			operationName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		// 开始新的span
		var span opentracing.Span
		if spanCtx != nil {
			span = tracer.StartSpan(operationName, opentracing.ChildOf(spanCtx))
		} else {
			span = tracer.StartSpan(operationName)
		}
		defer span.Finish()

		// 设置span标签
		ext.HTTPMethod.Set(span, c.Request.Method)
		ext.HTTPUrl.Set(span, c.Request.URL.String())
		ext.Component.Set(span, "gin-http")
		span.SetTag("http.remote_addr", c.ClientIP())
		span.SetTag("http.user_agent", c.Request.UserAgent())

		// 添加请求ID到span
		if requestID := c.GetString("request_id"); requestID != "" {
			span.SetTag("request.id", requestID)
		}

		// 将span上下文存储到gin上下文中
		c.Set("tracing_span", span)
		c.Set("tracing_context", span.Context())

		// 处理请求
		c.Next()

		// 设置响应状态码
		statusCode := c.Writer.Status()
		ext.HTTPStatusCode.Set(span, uint16(statusCode))

		// 如果是错误状态码，标记为错误
		if statusCode >= 400 {
			ext.Error.Set(span, true)
			span.LogFields(
				log.String("event", "error"),
				log.Int("status_code", statusCode),
			)
		}

		// 记录错误信息
		if len(c.Errors) > 0 {
			ext.Error.Set(span, true)
			for _, err := range c.Errors {
				span.LogFields(
					log.String("event", "error"),
					log.String("message", err.Error()),
				)
			}
		}
	}
}

// GetSpanFromContext 从gin上下文中获取span
func GetSpanFromContext(c *gin.Context) opentracing.Span {
	if span, exists := c.Get("tracing_span"); exists {
		if s, ok := span.(opentracing.Span); ok {
			return s
		}
	}
	return nil
}

// GetSpanContextFromContext 从gin上下文中获取span上下文
func GetSpanContextFromContext(c *gin.Context) opentracing.SpanContext {
	if spanCtx, exists := c.Get("tracing_context"); exists {
		if sc, ok := spanCtx.(opentracing.SpanContext); ok {
			return sc
		}
	}
	return nil
}

// StartChildSpan 在当前请求上下文中开始一个子span
func StartChildSpan(c *gin.Context, operationName string) opentracing.Span {
	parentSpan := GetSpanFromContext(c)
	if parentSpan != nil {
		return opentracing.StartSpan(operationName, opentracing.ChildOf(parentSpan.Context()))
	}
	return opentracing.StartSpan(operationName)
}
