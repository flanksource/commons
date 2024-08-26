package middlewares

import (
	"net/http"

	"github.com/flanksource/commons/logger"

	"github.com/henvic/httpretty"
)

func NewLogger(config TraceConfig) Middleware {
	l := &httpretty.Logger{
		Time:           config.Timing,
		TLS:            config.TLS,
		RequestHeader:  config.Headers,
		RequestBody:    config.Body,
		ResponseHeader: config.ResponseHeaders,
		ResponseBody:   config.Body,
		Colors:         true, // erase line if you don't like colors
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}

	l.SkipHeader(logger.SensitiveHeaders)
	return func(rt http.RoundTripper) http.RoundTripper {
		return logger.NewHttpLogger(logger.GetLogger(), rt)
	}
}
