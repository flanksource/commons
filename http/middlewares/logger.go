package middlewares

import (
	"net/http"

	"github.com/henvic/httpretty"
)

func NewLogger(config TraceConfig) Middleware {
	logger := &httpretty.Logger{
		Time:           config.Timing,
		TLS:            config.TLS,
		RequestHeader:  config.Headers,
		RequestBody:    config.Body,
		ResponseHeader: config.ResponseHeaders,
		ResponseBody:   config.Body,
		Colors:         true, // erase line if you don't like colors
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}
	return func(rt http.RoundTripper) http.RoundTripper {
		return logger.RoundTripper(rt)
	}

}
