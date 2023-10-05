package context

import (
	gocontext "context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/kubernetes"
)

// TODO:
type Context interface {
	Namespace() string
	Kubernetes() kubernetes.Interface

	WithContext(ctx gocontext.Context) Context
	WithTimeout(timeout time.Duration) (Context, func())

	Errorf(format string, args ...any)

	StartTrace(tracerName string, spanName string) (Context, trace.Span)
	SetSpanAttributes(attrs ...attribute.KeyValue)
}
