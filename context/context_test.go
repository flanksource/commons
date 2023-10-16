package context

import (
	gocontext "context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	. "github.com/onsi/gomega"
)

var exporter *stdouttrace.Exporter

func init() {

	var err error
	exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(resource.NewSchemaless()),
	)

	tracer = provider.Tracer("example.com/basic")

}

var tracer trace.Tracer

func TestContext(t *testing.T) {
	RegisterTestingT(t)
	ctx := NewContext(
		gocontext.Background(),
		WithTracer(tracer),
		WithDebugFn(func(Context) bool {
			return true
		}),
		WithTraceFn(func(Context) bool {
			return true
		}),
	)

	Expect(ctx.Value("process")).To(BeNil())

	outer := ctx.WithValue("process", "outer")
	Expect(outer.Value("process")).To(Equal("outer"))

	t.Log("Test_Context")

	ctx, span := outer.StartSpan("outer")
	defer span.End()
	span.SetAttributes(attribute.String("process", "outer"))

	ctx.Debugf("Debug message")
	ctx.Tracef("Trace message")

	inner, innerSpan := ctx.StartSpan("inner")
	innerSpan.SetAttributes(attribute.String("process", "inner"))
	Expect(inner.Value("process")).To(Equal("outer"))
	inner = inner.WithValue("process", "inner")
	Expect(inner.Value("process")).To(Equal("inner"))
	Expect(outer.Value("process")).To(Equal("outer"))

	inner.Debugf("Debug message from inner")
	inner.Tracef("Trace message from inner")
	defer innerSpan.End()

	span.AddEvent("test event from outer")

}
