package core

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

const JaegerTracerCTX = "TracerCTX"

var JaegerTracer *JaegerTracingSt = nil

type JaegerTracingSt struct {
	jConfig   jaegercfg.Configuration
	jReporterConfig jaegercfg.ReporterConfig
	jReporter jaeger.Reporter
	JTracer  opentracing.Tracer
	JCloser  io.Closer
}

//初始化全局的tracing对象处理逻辑
func IniJaegerTracing(serviceName, agentHost string) *JaegerTracingSt {
	if JaegerTracer == nil {//初始化的情况
		JaegerTracer = &JaegerTracingSt{jConfig: jaegercfg.Configuration{
			Sampler: &jaegercfg.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			ServiceName: serviceName,
		}, jReporterConfig: jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: agentHost,
		}}
		JaegerTracer.jReporter, _ = JaegerTracer.jReporterConfig.NewReporter(serviceName, jaeger.NewNullMetrics(), jaeger.NullLogger)
		JaegerTracer.JTracer, JaegerTracer.JCloser, _ = JaegerTracer.jConfig.NewTracer(jaegercfg.Reporter(JaegerTracer.jReporter))
	}
	return JaegerTracer
}

//获取跟踪数据上下文信息
func GetTracingCtx(c *gin.Context) context.Context {
	if oCtx, isExist := c.Get("ctx"); isExist {
		if ctx, ok := oCtx.(context.Context); ok {
			return ctx
		}
	}
	return nil
}

func GINTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		if JaegerTracer != nil {
			path := c.Request.URL.Path
			span := JaegerTracer.JTracer.StartSpan(path, ext.SpanKindRPCServer)
			ext.HTTPUrl.Set(span, path)
			ext.HTTPMethod.Set(span, c.Request.Method)
			traceCtx := opentracing.ContextWithSpan(context.Background(), span)
			c.Set(JaegerTracerCTX, traceCtx)
			c.Next()
			ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
			span.Finish()
		} else {
			c.Next()
		}
	}
}