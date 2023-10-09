package context

import (
	gocontext "context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Context interface {
	Errorf(err error)
	StartSpan(tracer trace.Tracer, spanName string) trace.Span
	SetSpanAttributes(key string, val any)
}

func NewContext() Context {
	return &context{}
}

// context implements Context
type context struct {
	gocontext.Context
}

func (c *context) StartSpan(tracer trace.Tracer, spanName string) trace.Span {
	traceCtx, span := tracer.Start(c.Context, spanName)
	c.Context = traceCtx
	return span
}

func (c *context) SetSpanAttributes(key string, val any) {
	var attr attribute.Value
	switch v := val.(type) {
	case bool:
		attr = attribute.BoolValue(v)
	case string:
		attr = attribute.StringValue(v)
	case int:
		attr = attribute.IntValue(v)
	case int32:
		attr = attribute.IntValue(int(v))
	case int64:
		attr = attribute.Int64Value(v)
	case float64:
		attr = attribute.Float64Value(v)
	case []bool:
		attr = attribute.BoolSliceValue(v)
	case []int64:
		attr = attribute.Int64SliceValue(v)
	case []float64:
		attr = attribute.Float64SliceValue(v)
	case []string:
		attr = attribute.StringSliceValue(v)
	default:
		attr = attribute.StringValue(fmt.Sprintf("%v", v))
	}

	trace.SpanFromContext(c).SetAttributes(attribute.KeyValue{Key: attribute.Key(key), Value: attr})
}

func (c *context) Errorf(err error) {
	span := trace.SpanFromContext(c)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
