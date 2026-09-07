package har

import (
	"bytes"
	"encoding/json"
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

// NewMiddleware returns a middlewares.Middleware that captures each request/response
// pair into a *Entry and calls handler. If handler is nil, the middleware is a no-op.
func NewMiddleware(cfg HARConfig, handler func(*Entry)) middlewares.Middleware {
	if handler == nil {
		return func(next http.RoundTripper) http.RoundTripper {
			return next
		}
	}
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return capture(req, next, cfg, handler)
		})
	}
}

func capture(req *http.Request, next http.RoundTripper, cfg HARConfig, handler func(*Entry)) (*http.Response, error) {
	started := time.Now()

	entry := &Entry{
		StartedDateTime: started.UTC().Format(time.RFC3339),
		Request:         buildRequest(req, cfg),
	}

	waitStart := time.Now()
	resp, err := next.RoundTrip(req)
	waitMs := float64(time.Since(waitStart).Microseconds()) / 1000.0

	entry.Timings = Timings{Wait: waitMs}
	entry.Time = waitMs

	if resp != nil {
		entry.Response = buildResponse(resp, cfg)
	}

	handler(entry)
	return resp, err
}

// CaptureRedirect builds a HAR entry from a redirect hop's request and response.
func CaptureRedirect(req *http.Request, resp *http.Response, cfg HARConfig) *Entry {
	return &Entry{
		StartedDateTime: time.Now().UTC().Format(time.RFC3339),
		Request:         buildRequest(req, cfg),
		Response:        buildResponse(resp, cfg),
	}
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

func buildResponse(resp *http.Response, cfg HARConfig) Response {
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
	if resp.Body != nil && (shouldCapture(ct, cfg.CaptureContentTypes) || resp.StatusCode >= 400) {
		body, restored := readBody(resp.Body, cfg.MaxBodySize, resp.ContentLength)
		resp.Body = restored
		har.BodySize = body.totalSize
		har.Content = Content{
			Size:      body.totalSize,
			MimeType:  ct,
			Text:      harBody(body.text, ct, cfg),
			Truncated: body.truncated,
		}
	}

	return har
}

type bodyResult struct {
	text      string
	totalSize int64
	truncated bool
}

func readBody(r io.ReadCloser, maxSize, knownSize int64) (bodyResult, io.ReadCloser) {
	if maxSize <= 0 {
		all, _ := io.ReadAll(r)
		return bodyResult{
			text:      string(all),
			totalSize: int64(len(all)),
		}, &replayedBody{Reader: bytes.NewReader(all), Closer: r}
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
	}, &replayedBody{Reader: io.MultiReader(bytes.NewReader(prefix), r), Closer: r}
}

type replayedBody struct {
	io.Reader
	io.Closer
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
