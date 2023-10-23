package middlewares

import (
	"bytes"
	"io"
	netHttp "net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/flanksource/commons/http"

func NewTracedTransport(config TraceConfig) *traceTransport {
	return &traceTransport{
		tracer: otel.GetTracerProvider().Tracer(tracerName),
		Config: config,
	}
}

type TraceConfig struct {
	// SpanName is an optional name for the span.
	// If not provided, the hostname of the requesting URL will be used.
	SpanName string

	// A list of patterns for headers which should be redacted in the trace.
	RedactedHeaders []string

	// MaxBodyLength is the max size of the body, in bytes, that will be traced.
	// If the response body is larger than this, it will not be traced at all.
	//
	//  Default: 4096 (4MB)
	MaxBodyLength int64

	// Body controls whether the request Body is traced
	Body bool

	// Response controls whether the Response body is traced
	Response bool

	// ResponseHeaders controls whether the response headers are traced
	ResponseHeaders bool

	// QueryParam controls whether the query parameters are traced
	QueryParam bool

	// Headers controls whether the Headers are traced
	Headers bool

	// TLS connection information
	TLS bool

	Timing bool
}

type traceTransport struct {
	tracer trace.Tracer

	Config TraceConfig
}

func (t *traceTransport) TraceAll(val bool) *traceTransport {
	t.Config.Body = true
	t.Config.Response = true
	t.Config.ResponseHeaders = true
	t.Config.QueryParam = true
	t.Config.Headers = true
	t.Config.TLS = true
	t.Config.Timing = true
	return t
}

func (t *traceTransport) TraceBody(val bool) *traceTransport {
	t.Config.Body = val
	return t
}

func (t *traceTransport) TraceResponse(val bool) *traceTransport {
	t.Config.Response = val
	return t
}

func (t *traceTransport) TraceResponseHeaders(val bool) *traceTransport {
	t.Config.ResponseHeaders = val
	return t
}

func (t *traceTransport) TraceQueryParam(val bool) *traceTransport {
	t.Config.QueryParam = val
	return t
}

func (t *traceTransport) TraceHeaders(val bool) *traceTransport {
	t.Config.Headers = val
	return t
}

func (t *traceTransport) RedactHeaders(patterns []string) *traceTransport {
	t.Config.RedactedHeaders = patterns
	return t
}

func (t *traceTransport) MaxBodyLength(val int64) *traceTransport {
	t.Config.MaxBodyLength = val
	return t
}

// SpanName sets the name of the span.
// If not provided, the hostname of the requesting URL will be used.
func (t *traceTransport) SpanName(val string) *traceTransport {
	t.Config.SpanName = val
	return t
}

func (t *traceTransport) TraceProvider(provider trace.TracerProvider) *traceTransport {
	t.tracer = provider.Tracer(tracerName)
	return t
}

func (t *traceTransport) RoundTripper(rt netHttp.RoundTripper) netHttp.RoundTripper {
	return RoundTripperFunc(func(ogRequest *netHttp.Request) (*netHttp.Response, error) {
		// According to RoundTripper spec, we shouldn't modify the origin request.
		req := ogRequest.Clone(ogRequest.Context())

		propagator := propagation.TraceContext{}
		propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))

		spanName := t.Config.SpanName
		if spanName == "" {
			spanName = req.URL.Host
		}

		_, span := t.tracer.Start(req.Context(), "http-"+spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("request.method", req.Method),
			attribute.String("request.url", req.URL.String()),
			attribute.String("request.host", req.Host),
		)

		if t.Config.Headers {
			for key, values := range SanitizeHeaders(req.Header, t.Config.RedactedHeaders...) {
				for _, value := range values {
					span.SetAttributes(attribute.String("request.header."+key, value))
				}
			}
		}

		span.SetAttributes(attribute.Int64("request.content-length", req.ContentLength))
		if req.Body != nil && t.Config.Body {
			if b, err := io.ReadAll(req.Body); err == nil {
				span.SetAttributes(attribute.String("request.body", string(b)))
				req.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}

		if t.Config.QueryParam && req.URL.RawQuery != "" {
			for q, val := range req.URL.Query() {
				span.SetAttributes(attribute.StringSlice("request.query."+q, val))
			}
		}

		resp, err := rt.RoundTrip(req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		if t.Config.ResponseHeaders {
			for key, values := range SanitizeHeaders(resp.Header, t.Config.RedactedHeaders...) {
				for _, value := range values {
					span.SetAttributes(attribute.String("response.header."+key, value))
				}
			}
		}

		if t.Config.Response {
			if b, err := io.ReadAll(io.LimitReader(resp.Body, t.Config.MaxBodyLength)); err == nil {
				span.SetAttributes(attribute.String("response.body", string(b)))
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}

		span.SetAttributes(attribute.String("response.status", resp.Status))
		span.SetAttributes(attribute.Int64("response.content-length", resp.ContentLength))

		return resp, nil
	})
}
