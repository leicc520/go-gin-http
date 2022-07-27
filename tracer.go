package core

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func init() {
	jcfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		ServiceName: "serviceName",
	}

	report := jaegercfg.ReporterConfig{
		LogSpans:           true,
		LocalAgentHostPort: "locahost:6831",
	}

	reporter, _ := report.NewReporter(serviceName, jaeger.NewNullMetrics(), jaeger.NullLogger)
	tracer, closer, _ := jcfg.NewTracer(
		jaegercfg.Reporter(reporter),
	)
	//ctx = gtx.Get("ctx").(context.Context)
}

func GINTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		span := tracer.StartSpan(path, ext.SpanKindRPCServer)
		ext.HTTPUrl.Set(span, path)
		ext.HTTPMethod.Set(span, ctx.Request.Method)
		traceCtx := opentracing.ContextWithSpan(context.Background(), span)
		c.Set("ctx", traceCtx)
		c.Next()
		ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
		span.Finish()
	}
}