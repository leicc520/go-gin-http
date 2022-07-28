package core

import (
	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-core/tracing"
	"github.com/leicc520/go-orm/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

//链路追踪中间件处理逻辑
func GINTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tracing.IsTracing() {
			c.Next() //不需要做链路跟踪的情况
			return
		}
		var span opentracing.Span
		spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			span  = opentracing.GlobalTracer().StartSpan(c.Request.URL.Path, ext.SpanKindRPCServer)
		} else {
			span  = opentracing.StartSpan(c.Request.URL.Path, opentracing.ChildOf(spCtx), ext.SpanKindRPCServer)
		}
		span.SetTag("http.signature", JWTACLToken(c))
		ext.HTTPUrl.Set(span, c.Request.URL.Path)
		ext.HTTPMethod.Set(span, c.Request.Method)
		//关闭链路追踪的句柄数据信息
		defer func() {
			span.Finish()
			log.Write(log.INFO, c.Request.RequestURI, " tracing ....")
		}()
		c.Set(tracing.JaegerSpanCTX, span.Context())
		c.Next()
		ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
	}
}