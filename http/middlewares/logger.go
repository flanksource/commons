package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/console"
	commonsCtx "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/logger/httpretty"
)

type jsonFormatter struct{}

func (j *jsonFormatter) Match(mediatype string) bool {
	return strings.Contains(mediatype, "json")
}

func (j *jsonFormatter) Format(w io.Writer, src []byte) error {
	var m map[string]any
	if err := json.Unmarshal(src, &m); err != nil {
		return err
	}
	sanitized := logger.StripSecretsFromMap(m)
	b, err := json.MarshalIndent(sanitized, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprint(w, api.CodeBlock("json", string(b)).ANSI())
	return nil
}

type formURLEncodedFormatter struct{}

func (f *formURLEncodedFormatter) Match(mediatype string) bool {
	return mediatype == "application/x-www-form-urlencoded"
}

func (f *formURLEncodedFormatter) Format(w io.Writer, src []byte) error {
	values, err := url.ParseQuery(string(src))
	if err != nil {
		return err
	}
	m := make(map[string]string)
	for k, v := range values {
		joined := strings.Join(v, ",")
		if logger.IsSensitiveKey(k) {
			joined = logger.PrintableSecret(joined)
		}
		m[k] = joined
	}
	fmt.Fprint(w, clicky.Map(m).ANSI())
	return nil
}

func getLogger(req *http.Request) logger.Logger {
	if req == nil || req.Context() == nil {
		return logger.GetLogger()
	}
	return commonsCtx.LoggerFromContext(req.Context())
}

func isSensitiveHeader(key string) bool {
	for _, h := range logger.SensitiveHeaders {
		if strings.EqualFold(h, key) {
			return true
		}
	}
	return false
}

func headerMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, v := range h {
		joined := strings.Join(v, ", ")
		if isSensitiveHeader(k) {
			m[k] = logger.PrintableSecret(joined)
		} else {
			m[k] = joined
		}
	}
	return m
}

func readBody(body io.ReadCloser) (string, io.ReadCloser) {
	if body == nil {
		return "", nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return "", io.NopCloser(bytes.NewReader(data))
	}
	return string(data), io.NopCloser(bytes.NewReader(data))
}

func sanitizeBody(body string) any {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err == nil {
		return logger.StripSecretsFromMap(m)
	}
	if values, err := url.ParseQuery(body); err == nil && len(values) > 0 {
		sanitized := make(map[string]string, len(values))
		for k := range values {
			v := values.Get(k)
			if logger.IsSensitiveKey(k) {
				sanitized[k] = logger.PrintableSecret(v)
			} else {
				sanitized[k] = v
			}
		}
		return sanitized
	}
	return body
}

func newContextLogger(config TraceConfig, verbose logger.Verbose) Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if logger.IsJsonLogs() {
				return jsonLogger(config, verbose, rt, req)
			}
			return prettyLogger(config, verbose, rt, req)
		})
	}
}

func logAt(verbose logger.Verbose, req *http.Request, level int, format string, args ...interface{}) {
	if verbose != nil {
		getLogger(req).V(level).Always().Infof(format, args...)
	} else {
		getLogger(req).Infof(format, args...)
	}
}

func jsonLogAt(verbose logger.Verbose, req *http.Request, level int, kv []interface{}, format string, args ...interface{}) {
	if verbose != nil {
		getLogger(req).V(level).Always().WithValues(kv...).Infof(format, args...)
	} else {
		getLogger(req).WithValues(kv...).Infof(format, args...)
	}
}

func verbosityLevel(config TraceConfig) int {
	if config.Body || config.Response {
		return 3
	}
	if config.Headers || config.ResponseHeaders {
		return 2
	}
	return 1
}

func jsonLogger(config TraceConfig, verbose logger.Verbose, rt http.RoundTripper, req *http.Request) (*http.Response, error) {
	var reqBody string
	if config.Body && req.Body != nil {
		reqBody, req.Body = readBody(req.Body)
	}

	start := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsed := time.Since(start)
	level := verbosityLevel(config)

	kv := []interface{}{
		"method", req.Method,
		"url", req.URL.String(),
	}

	if config.Timing {
		kv = append(kv, "duration", elapsed.Truncate(time.Millisecond).String())
	}

	if config.Headers {
		kv = append(kv, "headers", headerMap(req.Header))
	}
	if config.Body && reqBody != "" {
		kv = append(kv, "body", sanitizeBody(reqBody))
	}

	if err != nil {
		kv = append(kv, "error", err.Error())
		jsonLogAt(verbose, req, level, kv, "%s %s error %s", req.Method, req.URL, elapsed.Truncate(time.Millisecond))
		return nil, err
	}

	kv = append(kv, "status", resp.StatusCode)

	if config.ResponseHeaders {
		kv = append(kv, "responseHeaders", headerMap(resp.Header))
	}
	if config.Response && resp.Body != nil {
		var respBody string
		respBody, resp.Body = readBody(resp.Body)
		if respBody != "" {
			kv = append(kv, "responseBody", sanitizeBody(respBody))
		}
	}

	jsonLogAt(verbose, req, level, kv, "%s %s %d %s", req.Method, req.URL, resp.StatusCode, elapsed.Truncate(time.Millisecond))
	return resp, nil
}

func prettyLogger(config TraceConfig, verbose logger.Verbose, rt http.RoundTripper, req *http.Request) (*http.Response, error) {
	var buf bytes.Buffer
	l := &httpretty.Logger{
		TLS:            config.TLS,
		RequestHeader:  config.Headers,
		RequestBody:    config.Body,
		ResponseHeader: config.ResponseHeaders,
		ResponseBody:   config.Response,
		Auth:           config.Auth,
		Colors:         true,
		Formatters:     []httpretty.Formatter{&jsonFormatter{}, &formURLEncodedFormatter{}},
	}
	l.SetOutput(&buf)
	inner := l.RoundTripper(rt)
	start := time.Now()
	resp, err := inner.RoundTrip(req)
	elapsed := time.Since(start)
	if buf.Len() > 0 {
		msg := buf.String()
		if config.Timing {
			suffix := ""
			if resp != nil {
				suffix = fmt.Sprintf(" %d", resp.StatusCode)
			} else if err != nil {
				suffix = " error"
			}
			suffix += fmt.Sprintf(" %s", elapsed.Truncate(time.Millisecond))

			lines := strings.Split(msg, "\n")
			for i, line := range lines {
				if strings.Contains(line, req.Method) {
					lines[i] = strings.TrimRight(line, "\r\n") + suffix
					break
				}
			}
			msg = strings.Join(lines, "\n")
		}
		logAt(verbose, req, verbosityLevel(config), strings.TrimSpace(msg))
	}
	return resp, err
}

func statusColor(code int) func(string, ...interface{}) string {
	if code >= 200 && code < 300 {
		return console.Greenf
	} else if code >= 400 {
		return console.Redf
	}
	return console.Yellowf
}

func newContextAccessLog(verbose logger.Verbose) Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := rt.RoundTrip(req)
			elapsed := time.Since(start)

			if logger.IsJsonLogs() {
				kv := []interface{}{"method", req.Method, "url", req.URL.String(), "duration", elapsed.Truncate(time.Millisecond).String()}
				if err != nil {
					kv = append(kv, "error", err.Error())
					jsonLogAt(verbose, req, 1, kv, "%s %s error %s", req.Method, req.URL, elapsed.Truncate(time.Millisecond))
					return nil, err
				}
				kv = append(kv, "status", resp.StatusCode)
				jsonLogAt(verbose, req, 1, kv, "%s %s %d %s", req.Method, req.URL, resp.StatusCode, elapsed.Truncate(time.Millisecond))
				return resp, nil
			}

			if err != nil {
				logAt(verbose, req, 1, "%s %s %s %s", console.Bluef("%s", req.Method), console.Yellowf("%s", req.URL), console.Redf("error"), elapsed.Truncate(time.Millisecond))
				return nil, err
			}
			logAt(verbose, req, 1, "%s %s %s %s", console.Bluef("%s", req.Method), console.Yellowf("%s", req.URL), statusColor(resp.StatusCode)("%d", resp.StatusCode), elapsed.Truncate(time.Millisecond))
			return resp, nil
		})
	}
}

func NewLogger(config TraceConfig, verbose ...logger.Verbose) Middleware {
	var v logger.Verbose
	if len(verbose) > 0 {
		v = verbose[0]
	}
	if config.AccessLog {
		return newContextAccessLog(v)
	}
	return newContextLogger(config, v)
}
