package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flanksource/commons/logger/httpretty"
	"github.com/flanksource/commons/properties"
)

type redactedJSONFormatter struct{}

func (redactedJSONFormatter) Match(mediaType string) bool {
	return strings.Contains(mediaType, "json")
}

func (redactedJSONFormatter) Format(w io.Writer, src []byte) error {
	redacted := []byte(StripSecrets(string(src)))
	if !json.Valid(redacted) {
		_, err := fmt.Fprint(w, string(redacted))
		return err
	}

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, redacted, "", "    "); err != nil {
		return err
	}
	_, err := w.Write(formatted.Bytes())
	return err
}

var SensitiveHeaders = []string{
	"Authorization",
	"Set-Cookie",
	"Cookie",
	"Proxy-Authorization",
	"Cookie",
}

const (
	HTTPLogResponseBodyLengthProperty = "http.log.response.body.length"
	defaultHTTPLogResponseBodyLength  = int64(4 * 1024)
)

// HTTPLogResponseBodyLength returns the configured response body log limit.
// fallback is used when the property is unset, invalid, or non-positive;
// non-positive fallbacks use the 4 KiB default. The limit is always positive:
// an unbounded limit would make httpretty buffer and log a whole response.
func HTTPLogResponseBodyLength(fallback int64) int64 {
	if fallback <= 0 {
		fallback = defaultHTTPLogResponseBodyLength
	}
	limit := int64(properties.Bytes(int(fallback), HTTPLogResponseBodyLengthProperty))
	if limit <= 0 {
		return fallback
	}
	return limit
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
		Time:            logger.IsLevelEnabled(5),
		TLS:             logger.IsLevelEnabled(6),
		Auth:            logger.IsLevelEnabled(6),
		RequestHeader:   logger.IsLevelEnabled(5),
		RequestBody:     logger.IsLevelEnabled(6),
		ResponseHeader:  logger.IsLevelEnabled(5),
		ResponseBody:    logger.IsLevelEnabled(7),
		Colors:          true, // erase line if you don't like colors
		Formatters:      []httpretty.Formatter{redactedJSONFormatter{}},
		MaxResponseBody: HTTPLogResponseBodyLength(0),
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
		Time:            logger.IsLevelEnabled(headerLevel),
		TLS:             logger.IsLevelEnabled(headerLevel),
		RequestHeader:   logger.IsLevelEnabled(headerLevel),
		RequestBody:     logger.IsLevelEnabled(bodyLevel),
		ResponseHeader:  logger.IsLevelEnabled(headerLevel),
		ResponseBody:    logger.IsLevelEnabled(bodyLevel),
		Auth:            logger.IsLevelEnabled(headerLevel),
		Colors:          true, // erase line if you don't like colors
		Formatters:      []httpretty.Formatter{redactedJSONFormatter{}},
		MaxResponseBody: HTTPLogResponseBodyLength(0),
	}

	l.SkipHeader(SensitiveHeaders)

	return l.RoundTripper(rt)
}
