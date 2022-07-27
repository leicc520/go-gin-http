package core
import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"log"
	"testing"
	"time"

	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func TestJaeger(t *testing.T) {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "127.0.0.1:6831", // 替换host
		},
	}

	closer, err := cfg.InitGlobalTracer(
		"go.test.srv",
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}

	var ctx = context.TODO()
	span1, ctx := opentracing.StartSpanFromContext(ctx, "span_1")
	time.Sleep(time.Second / 2)

	span11, _ := opentracing.StartSpanFromContext(ctx, "span_1-1")
	time.Sleep(time.Second / 2)
	span11.Finish()

	span1.Finish()

	defer closer.Close()
}
