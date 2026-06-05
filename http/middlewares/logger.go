package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
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
	"github.com/flanksource/commons/properties"
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

func headerMap(h http.Header, redactedHeaders ...string) map[string]string {
	h = logger.SanitizeHeaders(h, redactedHeaders...)
	m := make(map[string]string, len(h))
	for k, v := range h {
		m[k] = strings.Join(v, ", ")
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

func formParams(req *http.Request) (url.Values, bool) {
	if req == nil || req.Body == nil {
		return nil, false
	}
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return nil, false
	}
	body, restored := readBody(req.Body)
	req.Body = restored
	values, err := url.ParseQuery(body)
	if err != nil || len(values) == 0 {
		return nil, false
	}
	return values, true
}

func valueMap(values url.Values) map[string]string {
	m := make(map[string]string, len(values))
	for key, vals := range values {
		joined := strings.Join(vals, ",")
		if logger.IsSensitiveKey(key) {
			joined = logger.PrintableSecret(joined)
		}
		m[key] = joined
	}
	return m
}

func formatValueBlock(title string, values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:\n%s", title, clicky.Map(valueMap(values)).ANSI())
}

func accessURL(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	u := *req.URL
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func hasDetailedTrace(config TraceConfig) bool {
	return config.TLS ||
		config.Headers ||
		config.Body ||
		config.ResponseHeaders ||
		config.Response ||
		config.QueryParam ||
		config.FormParams ||
		config.Auth
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
	if config.Response {
		return 4
	}
	if config.Body || config.TLS {
		return 3
	}
	if config.Headers || config.ResponseHeaders || config.QueryParam || config.FormParams {
		return 2
	}
	return 1
}

func jsonLogger(config TraceConfig, verbose logger.Verbose, rt http.RoundTripper, req *http.Request) (*http.Response, error) {
	var reqBody string
	if config.Body && req.Body != nil {
		reqBody, req.Body = readBody(req.Body)
	}
	var form url.Values
	if config.FormParams && !config.Body {
		form, _ = formParams(req)
	}

	start := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsed := time.Since(start)
	level := verbosityLevel(config)

	kv := []interface{}{
		"method", req.Method,
		"url", accessURL(req),
	}

	if config.AccessLog || config.Timing {
		kv = append(kv, "duration", elapsed.Truncate(time.Millisecond).String())
	}

	if config.Headers {
		kv = append(kv, "headers", headerMap(req.Header, config.RedactedHeaders...))
	}
	if config.QueryParam && len(req.URL.Query()) > 0 {
		kv = append(kv, "query", valueMap(req.URL.Query()))
	}
	if len(form) > 0 {
		kv = append(kv, "form", valueMap(form))
	}
	if config.Body && reqBody != "" {
		kv = append(kv, "body", sanitizeBody(reqBody))
	}

	if err != nil {
		// Transport errors surface at INFO so a failed request is visible at -v=0.
		kv = append(kv, "error", err.Error())
		jsonLogAt(verbose, req, 0, kv, "%s %s error %s", req.Method, req.URL, elapsed.Truncate(time.Millisecond))
		return nil, err
	}

	kv = append(kv, "status", resp.StatusCode)

	if config.ResponseHeaders {
		kv = append(kv, "responseHeaders", headerMap(resp.Header, config.RedactedHeaders...))
	}
	if config.Response && resp.Body != nil {
		var respBody string
		respBody, resp.Body = readBody(resp.Body)
		if respBody != "" {
			kv = append(kv, "responseBody", sanitizeBody(respBody))
		}
	}

	if resp.StatusCode >= 400 {
		// Error responses surface at INFO with their body, even when the
		// configured trace level wouldn't otherwise capture the response body.
		if !config.Response {
			if body := readErrorBody(resp, config.MaxBodyLength); body != "" {
				kv = append(kv, "responseBody", sanitizeBody(body))
			}
		}
		jsonLogAt(verbose, req, 0, kv, "%s %s %d %s", req.Method, req.URL, resp.StatusCode, elapsed.Truncate(time.Millisecond))
		return resp, nil
	}

	// Error-only mode suppresses the success line (see logPrettyAccess).
	if config.AccessLogErrorsOnly {
		return resp, nil
	}
	jsonLogAt(verbose, req, level, kv, "%s %s %d %s", req.Method, req.URL, resp.StatusCode, elapsed.Truncate(time.Millisecond))
	return resp, nil
}

func prettyLogger(config TraceConfig, verbose logger.Verbose, rt http.RoundTripper, req *http.Request) (*http.Response, error) {
	var form url.Values
	if config.FormParams && !config.Body {
		form, _ = formParams(req)
	}

	detailed := hasDetailedTrace(config)
	start := time.Now()
	if !detailed {
		resp, err := rt.RoundTrip(req)
		elapsed := time.Since(start)
		logPrettyAccess(config, verbose, req, resp, err, elapsed)
		return resp, err
	}

	var buf bytes.Buffer
	l := &httpretty.Logger{
		TLS:             config.TLS,
		RequestHeader:   config.Headers,
		RequestBody:     config.Body,
		ResponseHeader:  config.ResponseHeaders,
		ResponseBody:    config.Response,
		Auth:            config.Auth,
		Colors:          true,
		Formatters:      []httpretty.Formatter{&jsonFormatter{}, &formURLEncodedFormatter{}},
		MaxRequestBody:  config.MaxBodyLength,
		MaxResponseBody: config.MaxBodyLength,
		RedactedHeaders: append(config.RedactedHeaders, logger.CommonRedactedHeaders...),
	}
	l.SetOutput(&buf)
	inner := l.RoundTripper(rt)
	resp, err := inner.RoundTrip(req)
	elapsed := time.Since(start)
	logPrettyAccess(config, verbose, req, resp, err, elapsed)
	if buf.Len() > 0 {
		msg := buf.String()
		var blocks []string
		if config.QueryParam {
			if block := formatValueBlock("Query Params", req.URL.Query()); block != "" {
				blocks = append(blocks, block)
			}
		}
		if block := formatValueBlock("Form Params", form); block != "" {
			blocks = append(blocks, block)
		}
		if len(blocks) > 0 {
			msg = strings.TrimSpace(msg) + "\n" + strings.Join(blocks, "\n")
		}
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

func logPrettyAccess(config TraceConfig, verbose logger.Verbose, req *http.Request, resp *http.Response, err error, elapsed time.Duration) {
	if !config.AccessLog {
		return
	}
	method := console.Bluef("%s", req.Method)
	url := console.Yellowf("%s", accessURL(req))
	dur := elapsed.Truncate(time.Millisecond)

	// Transport errors and responses >= 400 always log (with the response body),
	// so a failing request surfaces even when the access log is installed in
	// error-only mode at -v=0; the body captures the cause (e.g. an HTML 404/500
	// page) without raising verbosity.
	if err != nil {
		logAt(verbose, req, 0, "%s %s %s %s", method, url, console.Redf("error: %s", err.Error()), dur)
		return
	}
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if statusCode >= 400 {
		logAt(verbose, req, 0, "%s %s %s %s", method, url, statusColor(statusCode)("%d", statusCode), dur)
		if body := readErrorBody(resp, config.MaxBodyLength); body != "" {
			logAt(verbose, req, 0, "%s", body)
		}
		return
	}
	// Error-only mode (installed one level below base) suppresses the success
	// line; full access logging (base level and up) logs every request.
	if config.AccessLogErrorsOnly {
		return
	}
	logAt(verbose, req, 1, "%s %s %s %s", method, url, statusColor(statusCode)("%d", statusCode), dur)
}

// readErrorBody reads and restores resp.Body (so downstream consumers still see
// it), returning the body truncated to maxLen runes. maxLen <= 0 falls back to
// the http.log.response.body.length property (4KB default).
func readErrorBody(resp *http.Response, maxLen int64) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	limit := maxLen
	if limit <= 0 {
		limit = int64(properties.Int(4*1024, "http.log.response.body.length"))
	}
	body, restored := readBody(resp.Body)
	resp.Body = restored
	body = strings.TrimSpace(body)
	if int64(len(body)) > limit {
		return body[:limit] + "… (truncated)"
	}
	return body
}

func statusColor(code int) func(string, ...interface{}) string {
	if code >= 200 && code < 300 {
		return console.Greenf
	} else if code >= 400 {
		return console.Redf
	}
	return console.Yellowf
}

func NewLogger(config TraceConfig, verbose ...logger.Verbose) Middleware {
	var v logger.Verbose
	if len(verbose) > 0 {
		v = verbose[0]
	}
	return newContextLogger(config, v)
}
