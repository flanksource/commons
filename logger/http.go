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

// NewHttpLogger creates an HTTP logger that logs at predefined levels.
// Deprecated: Use NewHttpLoggerWithLevels for more control over logging levels.
//
// Default behavior:
//   - Headers and timing: Requires log level 5 (Trace3)
//   - Request body: Requires log level 6 (Trace4)
//   - Response body: Requires log level 7
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

// NewHttpLoggerWithLevels creates an HTTP logger with configurable log levels for headers and body.
//
// Parameters:
//   - logger: The logger instance to use
//   - rt: The underlying RoundTripper to wrap
//   - headerLevel: Minimum log level required to log headers, timing, and TLS info
//   - bodyLevel: Minimum log level required to log request/response bodies
//
// Example:
//
//	// Log headers at debug level (-v) and bodies at trace level (-vv)
//	transport := NewHttpLoggerWithLevels(logger, http.DefaultTransport, logger.Debug, logger.Trace)
func NewHttpLoggerWithLevels(logger Logger, rt http.RoundTripper, headerLevel, bodyLevel LogLevel) http.RoundTripper {
	if !logger.IsLevelEnabled(headerLevel) {
		return rt
	}

	l := &httpretty.Logger{
		Time:           logger.IsLevelEnabled(headerLevel),
		TLS:            logger.IsLevelEnabled(headerLevel),
		RequestHeader:  logger.IsLevelEnabled(headerLevel),
		RequestBody:    logger.IsLevelEnabled(bodyLevel),
		ResponseHeader: logger.IsLevelEnabled(headerLevel),
		ResponseBody:   logger.IsLevelEnabled(bodyLevel),
		Colors:         true, // erase line if you don't like colors
		Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
	}

	l.SkipHeader(SensitiveHeaders)

	return l.RoundTripper(rt)
}
