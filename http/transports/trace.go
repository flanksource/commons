package transports

import (
	"bytes"
	"io"
	"net/http"

	"github.com/flanksource/commons/bitmask"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TraceMode bitmask.Bits

const (
	TraceBody TraceMode = 1 << iota
	TraceResponse
	TracerQueryParam
	// ... Add more
)

func NewTracedTransport(tracer trace.Tracer) *traceTransport {
	return &traceTransport{
		tracer: tracer,
	}
}

type traceTransport struct {
	rt     http.RoundTripper
	tracer trace.Tracer
	mode   TraceMode
}

func (t *traceTransport) Wrap(next http.RoundTripper) http.RoundTripper {
	t.rt = next
	return t
}

func (t *traceTransport) Mode(m TraceMode) *traceTransport {
	t.mode = m
	return t
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_, span := t.tracer.Start(req.Context(), req.URL.Host)
	defer span.End()

	span.SetAttributes(attribute.String("request.method", req.Method))
	span.SetAttributes(attribute.String("request.host", req.Host))

	if req.Body != nil && bitmask.Has(bitmask.Bits(t.mode), bitmask.Bits(TraceBody)) {
		b, _ := io.ReadAll(req.Body)
		span.SetAttributes(attribute.String("request.body", string(b)))

		req.Body = io.NopCloser(bytes.NewBuffer(b))
	}

	if bitmask.Has(bitmask.Bits(t.mode), bitmask.Bits(TracerQueryParam)) {
		span.SetAttributes(attribute.String("request.query", req.URL.RawQuery))
	}

	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if bitmask.Has(bitmask.Bits(t.mode), bitmask.Bits(TraceResponse)) {
		b, _ := io.ReadAll(resp.Body)
		span.SetAttributes(attribute.String("response.body", string(b)))

		resp.Body = io.NopCloser(bytes.NewBuffer(b))
	}

	span.SetAttributes(attribute.String("response.status", resp.Status))

	return resp, nil
}
