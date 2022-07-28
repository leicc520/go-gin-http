package core

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-orm/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

const JaegerTracer    = "JTracer"
const JaegerTracerCTX = "JTracerCTX"

var gTracer *JaegerTracingSt = nil

type JaegerTracingConfigSt struct {
	Agent     string  `json:"agent"`
	Type      string  `json:"type"`
	Param     float64 `json:"param"`
	IsTrace   bool    `json:"is_trace"`
	mu        sync.RWMutex
}

type JaegerTracingSt struct {
	jConfig   jaegercfg.Configuration
	jReporterConfig jaegercfg.ReporterConfig
	jReporter jaeger.Reporter
	Config    *JaegerTracingConfigSt
}

//设置是否开启链路追踪
func (s *JaegerTracingConfigSt) IsTracing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.IsTrace && len(s.Agent) > 0 && len(s.Type) > 0 {
		return true
	}
	return false
}

//设置开启链路追踪
func (s *JaegerTracingConfigSt) SetTracing(isTrace bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsTrace = isTrace
}

//初始化全局的tracing对象处理逻辑
func (s *JaegerTracingConfigSt) Init(serviceName string) *JaegerTracingSt {
	//数据为空 且配置不为空的 情况逻辑
	if gTracer == nil && len(s.Type) > 0 && len(s.Agent) > 0 {
		gTracer = &JaegerTracingSt {
			jConfig: jaegercfg.Configuration {
				Sampler: &jaegercfg.SamplerConfig {
					Type:  s.Type,
					Param: s.Param,
				},
				ServiceName: serviceName,
			},
			jReporterConfig: jaegercfg.ReporterConfig {
				LogSpans:           true,
				LocalAgentHostPort: s.Agent,
			},
			Config: s,
		}
		gTracer.jReporter, _ = gTracer.jReporterConfig.NewReporter(serviceName, jaeger.NewNullMetrics(), jaeger.NullLogger)
	}
	return gTracer
}

//获取跟踪数据上下文信息
func GetTracingCtx(c *gin.Context) context.Context {
	if oCtx, isExist := c.Get(JaegerTracerCTX); isExist {
		if ctx, ok := oCtx.(context.Context); ok {
			return ctx
		}
	}
	return nil
}

//链路追踪中间件处理逻辑
func GINTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		if gTracer == nil || !gTracer.Config.IsTracing() {
			c.Next() //不需要做链路跟踪的情况
			return
		}
		path := c.Request.URL.Path
		tracer, closer, err := gTracer.jConfig.NewTracer(jaegercfg.Reporter(gTracer.jReporter))
		if err != nil {//如果不为空说明成功
			log.Write(log.ERROR, "tracing error", err)
			c.Next()
			return
		}
		var span opentracing.Span
		spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			span  = tracer.StartSpan(path, ext.SpanKindRPCServer)
		} else {
			span  = opentracing.StartSpan(c.Request.URL.Path, opentracing.ChildOf(spCtx), ext.SpanKindRPCServer)
		}
		span.SetTag("http.signature", JWTACLToken(c))
		ext.HTTPUrl.Set(span, path)
		ext.HTTPMethod.Set(span, c.Request.Method)
		//关闭链路追踪的句柄数据信息
		defer func() {
			span.Finish()
			closer.Close()
		}()
		c.Set(JaegerTracer, tracer)
		c.Set(JaegerTracerCTX, span.Context())
		c.Next()
		ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
	}
}