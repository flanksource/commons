package http

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.GetTracerProvider().Tracer("commons-http-client")

func Error(span trace.Span, err error) bool {
	if err == nil {
		return false
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return true
}
