package har

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
)

// startedDateTimeLayout is RFC 3339 with fixed millisecond precision, so a
// client can tick the elapsed time of a pending entry.
const startedDateTimeLayout = "2006-01-02T15:04:05.000Z07:00"

// recordFunc registers the entry for a request that is about to be sent and
// returns the function that receives its completed form.
type recordFunc func(started time.Time, request Entry) (complete func(*Entry))

// captureFunc performs one round trip through next, recording it via record.
type captureFunc func(req *http.Request, next http.RoundTripper, cfg HARConfig, record recordFunc) (*http.Response, error)

// NewMiddleware returns a middlewares.Middleware that captures each request/response
// pair into a *Entry and calls handler once the request completes. If handler
// is nil, the middleware is a no-op. Use Collector.Middleware to also see
// requests while they are in flight.
func NewMiddleware(cfg HARConfig, handler func(*Entry)) middlewares.Middleware {
	return handlerMiddleware(capture, cfg, handler)
}

func handlerMiddleware(fn captureFunc, cfg HARConfig, handler func(*Entry)) middlewares.Middleware {
	if handler == nil {
		return func(next http.RoundTripper) http.RoundTripper {
			return next
		}
	}
	record := func(time.Time, Entry) func(*Entry) { return handler }
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return fn(req, next, cfg, record)
		})
	}
}

func capture(req *http.Request, next http.RoundTripper, cfg HARConfig, record recordFunc) (*http.Response, error) {
	started := time.Now()
	entry := &Entry{
		StartedDateTime: started.UTC().Format(startedDateTimeLayout),
		Request:         buildRequest(req, cfg),
	}
	complete := record(started, *entry)

	resp, err := next.RoundTrip(req)
	entry.Timings.Wait = millis(time.Since(started))

	var bodyErr error
	if resp != nil {
		receiveStart := time.Now()
		entry.Response, bodyErr = buildResponse(resp, cfg)
		entry.Timings.Receive = millis(time.Since(receiveStart))
	}
	entry.Time = entry.Timings.Wait + entry.Timings.Receive
	if failure := errors.Join(err, bodyErr); failure != nil {
		entry.Error = failure.Error()
	}

	complete(entry)
	return resp, err
}

func buildRequest(req *http.Request, cfg HARConfig) Request {
	har := Request{
		Method:      req.Method,
		URL:         harURL(req.URL, cfg),
		HTTPVersion: httpVersion(req.Proto),
		Cookies:     []Cookie{},
		Headers:     harHeaders(req.Header, cfg),
		QueryString: harQueryString(req.URL.Query(), cfg),
		HeadersSize: -1,
		BodySize:    -1,
	}

	ct := req.Header.Get("Content-Type")
	if req.Body != nil && shouldCapture(ct, cfg.CaptureContentTypes) {
		body, restored := readBody(req.Body, cfg.MaxBodySize, req.ContentLength)
		req.Body = restored
		har.BodySize = body.totalSize
		har.PostData = &PostData{
			MimeType: ct,
			Text:     harBody(body.text, ct, cfg),
		}
	}

	return har
}

// buildResponse captures the response and returns the error the body read hit,
// which the caller will also see when it reads the replayed body.
func buildResponse(resp *http.Response, cfg HARConfig) (Response, error) {
	har := Response{
		Status:      resp.StatusCode,
		StatusText:  http.StatusText(resp.StatusCode),
		HTTPVersion: httpVersion(resp.Proto),
		Cookies:     []Cookie{},
		Headers:     harHeaders(resp.Header, cfg),
		RedirectURL: "",
		HeadersSize: -1,
		BodySize:    -1,
	}

	ct := resp.Header.Get("Content-Type")
	if resp.Body == nil || (!shouldCapture(ct, cfg.CaptureContentTypes) && resp.StatusCode < 400) {
		return har, nil
	}
	body, restored := readBody(resp.Body, cfg.MaxBodySize, resp.ContentLength)
	resp.Body = restored
	har.BodySize = body.totalSize
	har.Content = Content{
		Size:      body.totalSize,
		MimeType:  ct,
		Text:      harBody(body.text, ct, cfg),
		Truncated: body.truncated,
	}
	return har, body.err
}

type bodyResult struct {
	text      string
	totalSize int64
	truncated bool
	// err is the error the capture read hit before EOF or the size limit.
	err error
}

// readBody captures up to maxSize bytes of r (all of it when maxSize <= 0) and
// returns a replacement body that replays them. When the capture read failed,
// the replacement yields the captured bytes and then that same error, so the
// caller's read fails exactly as it would have without capture.
func readBody(r io.ReadCloser, maxSize, knownSize int64) (bodyResult, io.ReadCloser) {
	if maxSize <= 0 {
		all, err := io.ReadAll(r)
		return bodyResult{
			text:      string(all),
			totalSize: int64(len(all)),
			err:       err,
		}, replayBody(r, all, err, nil)
	}

	limit := maxSize
	if maxSize < math.MaxInt64 {
		limit++
	}
	prefix, _ := io.ReadAll(io.LimitReader(r, limit))
	captured := prefix
	truncated := int64(len(prefix)) > maxSize
	if truncated {
		captured = prefix[:maxSize]
	}
	totalSize := int64(len(prefix))
	if truncated {
		totalSize = -1
		if knownSize > 0 {
			totalSize = knownSize
		}
	}

	return bodyResult{
		text:      string(captured),
		totalSize: totalSize,
		truncated: truncated,
		err:       err,
	}, replayBody(r, prefix, err, r)
}

// replayBody serves captured, then either readErr (when the capture failed) or
// unread (the rest of the stream, nil when the capture consumed all of it).
// Closing it closes body.
func replayBody(body io.ReadCloser, captured []byte, readErr error, unread io.Reader) io.ReadCloser {
	rest := unread
	if readErr != nil {
		rest = errReader{err: readErr}
	}
	if rest == nil {
		return &replayedBody{Reader: bytes.NewReader(captured), Closer: body}
	}
	return &replayedBody{Reader: io.MultiReader(bytes.NewReader(captured), rest), Closer: body}
}

type replayedBody struct {
	io.Reader
	io.Closer
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

func shouldCapture(contentType string, allowed []string) bool {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	ct = strings.TrimSpace(ct)
	for _, a := range allowed {
		if strings.HasPrefix(ct, strings.ToLower(a)) {
			return true
		}
	}
	return false
}

// harHeaders, harURL, harQueryString and harBody are the single gate through
// which every captured value passes. cfg.CaptureSensitive (-Phttp.har.sensitive)
// bypasses redaction so the archive can be replayed against the live API; by
// default credentials are masked with logger.PrintableSecret.
func harHeaders(headers http.Header, cfg HARConfig) []Header {
	if cfg.CaptureSensitive {
		return toHARHeaders(headers)
	}
	return toHARHeaders(logger.SanitizeHeaders(headers, cfg.RedactedHeaders...))
}

func harURL(u *url.URL, cfg HARConfig) string {
	if u == nil {
		return ""
	}
	if cfg.CaptureSensitive {
		return u.String()
	}
	return redactURL(u, cfg.RedactedBodyKeys)
}

func harQueryString(query url.Values, cfg HARConfig) []QueryString {
	qs := make([]QueryString, 0, len(query))
	for k, vs := range query {
		redact := !cfg.CaptureSensitive && isRedactedKey(k, cfg.RedactedBodyKeys)
		for _, v := range vs {
			if redact {
				v = logger.PrintableSecret(v)
			}
			qs = append(qs, QueryString{Name: k, Value: v})
		}
	}
	return qs
}

func harBody(text, contentType string, cfg HARConfig) string {
	if cfg.CaptureSensitive {
		return text
	}
	return redactBody(text, contentType, cfg.RedactedBodyKeys)
}

func redactBody(text, contentType string, extraKeys []string) string {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	ct = strings.TrimSpace(ct)

	switch ct {
	case "application/json":
		return redactJSON(text, extraKeys)
	case "application/x-www-form-urlencoded":
		return redactForm(text, extraKeys)
	default:
		return text
	}
}

func redactJSON(text string, extraKeys []string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		// A body that doesn't parse still gets key-based redaction, including
		// the caller's extraKeys, so a malformed payload can't leak a secret.
		return logger.StripSecrets(text, extraKeys...)
	}
	redacted := stripSecretMap(m, extraKeys)
	out, err := json.Marshal(redacted)
	if err != nil {
		return text
	}
	return string(out)
}

// stripSecretMap recursively redacts values whose key is sensitive (per
// logger.IsSensitiveKey or extraKeys), descending into nested objects and
// arrays so a sensitive field can't escape redaction by being nested, while
// preserving the rest of the structure.
func stripSecretMap(m map[string]any, extraKeys []string) map[string]any {
	clone := make(map[string]any, len(m))
	for k, v := range m {
		if isRedactedKey(k, extraKeys) {
			clone[k] = logger.PrintableSecret(fmt.Sprintf("%v", v))
			continue
		}
		clone[k] = stripSecretValue(v, extraKeys)
	}
	return clone
}

func stripSecretValue(v any, extraKeys []string) any {
	switch val := v.(type) {
	case map[string]any:
		return stripSecretMap(val, extraKeys)
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = stripSecretValue(e, extraKeys)
		}
		return out
	default:
		return v
	}
}

func redactForm(text string, extraKeys []string) string {
	vals, err := url.ParseQuery(text)
	if err != nil {
		return logger.StripSecrets(text, extraKeys...)
	}
	for k, vs := range vals {
		if isRedactedKey(k, extraKeys) {
			redacted := make([]string, len(vs))
			for i, v := range vs {
				redacted[i] = logger.PrintableSecret(v)
			}
			vals[k] = redacted
		}
	}
	return vals.Encode()
}

// isRedactedKey reports whether key is sensitive, either by the default
// logger.IsSensitiveKey heuristics or by a case-insensitive substring match
// against any of extraKeys.
func isRedactedKey(key string, extraKeys []string) bool {
	if logger.IsSensitiveKey(key) {
		return true
	}
	k := strings.ToLower(strings.TrimSpace(key))
	for _, e := range extraKeys {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" && strings.Contains(k, e) {
			return true
		}
	}
	return false
}

// redactURL renders u as a string with sensitive query-parameter values
// redacted. u itself is never mutated (it is still used for the live request).
func redactURL(u *url.URL, extraKeys []string) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	changed := false
	for k, vs := range q {
		if !isRedactedKey(k, extraKeys) {
			continue
		}
		for i := range vs {
			vs[i] = logger.PrintableSecret(vs[i])
		}
		q[k] = vs
		changed = true
	}
	if !changed {
		return u.String()
	}
	clone := *u
	clone.RawQuery = q.Encode()
	return clone.String()
}

func toHARHeaders(h http.Header) []Header {
	headers := make([]Header, 0, len(h))
	for name, vals := range h {
		for _, v := range vals {
			headers = append(headers, Header{Name: name, Value: v})
		}
	}
	return headers
}

func httpVersion(proto string) string {
	if proto == "" {
		return "HTTP/1.1"
	}
	return proto
}
