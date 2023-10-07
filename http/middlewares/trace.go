package middlewares

import (
	"bytes"
	"io"
	netHttp "net/http"

	"github.com/flanksource/commons/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/flanksource/commons/http"

func NewTracedTransport() *traceTransport {
	return &traceTransport{
		tracer: otel.GetTracerProvider().Tracer(tracerName),
	}
}

type traceTransport struct {
	// tracer is the creator of spans
	tracer trace.Tracer

	// body controls whether the request body is traced
	body bool

	// response controls whether the response body is traced
	response bool

	// responseHeaders controls whether the response headers are traced
	responseHeaders bool

	// queryParam controls whether the query parameters are traced
	queryParam bool

	// headers controls whether the headers are traced
	headers bool
}

func (t *traceTransport) TraceAll(val bool) *traceTransport {
	t.body = true
	t.response = true
	t.responseHeaders = true
	t.queryParam = true
	t.headers = true
	return t
}

func (t *traceTransport) TraceBody(val bool) *traceTransport {
	t.body = val
	return t
}

func (t *traceTransport) TraceResponse(val bool) *traceTransport {
	t.response = val
	return t
}

func (t *traceTransport) TraceResponseHeaders(val bool) *traceTransport {
	t.responseHeaders = val
	return t
}

func (t *traceTransport) TraceQueryParam(val bool) *traceTransport {
	t.queryParam = val
	return t
}

func (t *traceTransport) TraceHeaders(val bool) *traceTransport {
	t.headers = val
	return t
}

func (t *traceTransport) TraceProvider(provider trace.TracerProvider) *traceTransport {
	t.tracer = provider.Tracer(tracerName)
	return t
}

func (t *traceTransport) RoundTripper(rt netHttp.RoundTripper) netHttp.RoundTripper {
	return http.RoundTripperFunc(func(ogRequest *netHttp.Request) (*netHttp.Response, error) {
		// According to RoundTripper spec, we shouldn't modify the origin request.
		req := ogRequest.Clone(ogRequest.Context())

		propagator := propagation.TraceContext{}
		propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))

		_, span := t.tracer.Start(req.Context(), req.URL.Host)
		defer span.End()

		span.SetAttributes(
			attribute.String("request.method", req.Method),
			attribute.String("request.url", req.URL.String()),
			attribute.String("request.host", req.Host),
		)

		if t.headers {
			for key, values := range req.Header {
				for _, value := range values {
					span.SetAttributes(attribute.String("request.header."+key, value))
				}
			}
		}

		if req.Body != nil && t.body {
			if b, err := io.ReadAll(req.Body); err == nil {
				span.SetAttributes(attribute.String("request.body", string(b)))
				req.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}

		if t.queryParam && req.URL.RawQuery != "" {
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

		if t.responseHeaders {
			for key, values := range resp.Header {
				for _, value := range values {
					span.SetAttributes(attribute.String("response.header."+key, value))
				}
			}
		}

		if t.response {
			if b, err := io.ReadAll(resp.Body); err == nil {
				span.SetAttributes(attribute.String("response.body", string(b)))
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}

		span.SetAttributes(attribute.String("response.status", resp.Status))

		return resp, nil
	})
}
