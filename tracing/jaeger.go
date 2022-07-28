package tracing

import (
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-orm/log"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

const JaegerSpanCTX = "JSpanCTX"

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
	jTracer   opentracing.Tracer
	jCloser   io.Closer
	Config   *JaegerTracingConfigSt
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
func (s *JaegerTracingConfigSt) SetIsTracing(isTrace bool) {
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
		var err error = nil
		gTracer.jReporter, _ = gTracer.jReporterConfig.NewReporter(serviceName, jaeger.NewNullMetrics(), jaeger.NullLogger)
		gTracer.jTracer, gTracer.jCloser, err = gTracer.jConfig.NewTracer(jaegercfg.Reporter(gTracer.jReporter))
		if err != nil {//如果不为空说明成功
			log.Write(log.ERROR, "tracing error", err)
			gTracer = nil
		}
		opentracing.SetGlobalTracer(gTracer.jTracer)
	}
	return gTracer
}

//获取跟踪数据上下文信息
func GetTracingCtx(c *gin.Context) opentracing.SpanContext {
	if oCtx, isExist := c.Get(JaegerSpanCTX); isExist {
		if ctx, ok := oCtx.(opentracing.SpanContext); ok {
			return ctx
		}
	}
	return nil
}

//判定是否开启跟踪
func IsTracing() bool {
	if gTracer == nil || gTracer.Config == nil {
		return false
	}
	return gTracer.Config.IsTracing()
}

//关闭连接处理逻辑
func Close() {
	if gTracer != nil && gTracer.jCloser != nil {
		gTracer.jCloser.Close()
	}
}