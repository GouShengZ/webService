package tracer

import (
	"fmt"
	"io"

	"webservice/internal/config"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

// Init 初始化Jaeger链路追踪
func Init(cfg config.JaegerConfig) (io.Closer, error) {
	// 配置Jaeger
	jaegerCfg := jaegercfg.Configuration{
		ServiceName: cfg.ServiceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  cfg.SamplerType,
			Param: cfg.SamplerParam,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           false, // 禁用日志输出避免干扰
			LocalAgentHostPort: fmt.Sprintf("%s:%d", cfg.AgentHost, cfg.AgentPort),
		},
	}

	// 创建tracer
	tracer, closer, err := jaegerCfg.NewTracer(
		jaegercfg.Logger(jaegerlog.NullLogger), // 使用NullLogger避免日志干扰
		jaegercfg.Metrics(metrics.NullFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	// 设置全局tracer
	opentracing.SetGlobalTracer(tracer)

	return closer, nil
}

// StartSpan 开始一个新的span
func StartSpan(operationName string) opentracing.Span {
	return opentracing.StartSpan(operationName)
}

// StartSpanFromContext 从上下文开始一个新的span
func StartSpanFromContext(ctx opentracing.SpanContext, operationName string) opentracing.Span {
	return opentracing.StartSpan(operationName, opentracing.ChildOf(ctx))
}

// GetGlobalTracer 获取全局tracer
func GetGlobalTracer() opentracing.Tracer {
	return opentracing.GlobalTracer()
}
