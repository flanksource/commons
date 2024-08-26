package logger

import (
	"net/http"

	"github.com/henvic/httpretty"
)

var SensitiveHeaders = []string{
	"Authorization",
	"Set-Cookie",
	"Cookie",
	"Proxy-Authorization",
	"Cookie",
}

func NewHttpLogger(logger Logger, rt http.RoundTripper) http.RoundTripper {
	if !logger.IsLevelEnabled(5) {
		return rt
	}

	l := &httpretty.Logger{
		Time:           logger.IsLevelEnabled(5),
		TLS:            logger.IsLevelEnabled(5),
		RequestHeader:  logger.IsLevelEnabled(5),
		RequestBody:    logger.IsLevelEnabled(6),
		ResponseHeader: logger.IsLevelEnabled(5),
		ResponseBody:   logger.IsLevelEnabled(7),
		Colors:         true, // erase line if you don't like colors
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}

	l.SkipHeader(SensitiveHeaders)

	return l.RoundTripper(rt)
}
