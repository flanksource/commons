package har

import (
	"bytes"
	"encoding/json"
	"io"
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
		URL:         req.URL.String(),
		HTTPVersion: httpVersion(req.Proto),
		Cookies:     []Cookie{},
		Headers:     toHARHeaders(logger.SanitizeHeaders(req.Header, cfg.RedactedHeaders...)),
		QueryString: toQueryString(req.URL.Query()),
		HeadersSize: -1,
		BodySize:    -1,
	}

	ct := req.Header.Get("Content-Type")
	if req.Body != nil && shouldCapture(ct, cfg.CaptureContentTypes) {
		body, restored := readBody(req.Body, cfg.MaxBodySize)
		req.Body = restored
		har.BodySize = int64(len(body.raw))
		har.PostData = &PostData{
			MimeType: ct,
			Text:     redactBody(body.text, ct),
		}
	}

	return har
}

func buildResponse(resp *http.Response, cfg HARConfig) Response {
	har := Response{
		Status:      resp.StatusCode,
		StatusText:  resp.Status,
		HTTPVersion: httpVersion(resp.Proto),
		Cookies:     []Cookie{},
		Headers:     toHARHeaders(logger.SanitizeHeaders(resp.Header, cfg.RedactedHeaders...)),
		RedirectURL: "",
		HeadersSize: -1,
		BodySize:    -1,
	}

	ct := resp.Header.Get("Content-Type")
	if resp.Body != nil && (shouldCapture(ct, cfg.CaptureContentTypes) || resp.StatusCode >= 400) {
		body, restored := readBody(resp.Body, cfg.MaxBodySize)
		resp.Body = restored
		har.BodySize = body.totalSize
		har.Content = Content{
			Size:      body.totalSize,
			MimeType:  ct,
			Text:      redactBody(body.text, ct),
			Truncated: body.truncated,
		}
	}

	return har
}

type bodyResult struct {
	text      string
	raw       []byte
	totalSize int64
	truncated bool
}

func readBody(r io.ReadCloser, maxSize int64) (bodyResult, io.ReadCloser) {
	all, _ := io.ReadAll(r)
	_ = r.Close()

	total := int64(len(all))
	cap := all
	truncated := false

	if maxSize > 0 && total > maxSize {
		cap = all[:maxSize]
		truncated = true
	}

	return bodyResult{
		text:      string(cap),
		raw:       cap,
		totalSize: total,
		truncated: truncated,
	}, io.NopCloser(bytes.NewReader(all))
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

func redactBody(text, contentType string) string {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	ct = strings.TrimSpace(ct)

	switch ct {
	case "application/json":
		return redactJSON(text)
	case "application/x-www-form-urlencoded":
		return redactForm(text)
	default:
		return text
	}
}

func redactJSON(text string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return text
	}
	redacted := logger.StripSecretsFromMap(m)
	out, err := json.Marshal(redacted)
	if err != nil {
		return text
	}
	return string(out)
}

func redactForm(text string) string {
	vals, err := url.ParseQuery(text)
	if err != nil {
		return text
	}
	for k, vs := range vals {
		if logger.IsSensitiveKey(k) {
			redacted := make([]string, len(vs))
			for i, v := range vs {
				redacted[i] = logger.PrintableSecret(v)
			}
			vals[k] = redacted
		}
	}
	return vals.Encode()
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

func toQueryString(q url.Values) []QueryString {
	qs := make([]QueryString, 0, len(q))
	for k, vs := range q {
		for _, v := range vs {
			qs = append(qs, QueryString{Name: k, Value: v})
		}
	}
	return qs
}

func httpVersion(proto string) string {
	if proto == "" {
		return "HTTP/1.1"
	}
	return proto
}
