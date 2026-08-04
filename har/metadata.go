package har

import (
	"net/http"
	"time"

	"github.com/flanksource/commons/http/middlewares"
)

// NewMetadataMiddleware captures method, URL, sanitized headers, query string,
// status and timings — no request or response bodies. Body sizes use -1 per the
// HAR spec ("size unknown"). Use it when you want a HAR file for traffic
// analysis without paying the body-buffering cost.
//
// Ported from duty/connection/common.go's metadataHARMiddleware, which
// commons/http and commons-db each carried their own copy of.
func NewMetadataMiddleware(cfg HARConfig, handler func(*Entry)) middlewares.Middleware {
	if handler == nil {
		return func(next http.RoundTripper) http.RoundTripper {
			return next
		}
	}
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			entry := &Entry{
				StartedDateTime: time.Now().UTC().Format(time.RFC3339),
				Request: Request{
					Method:      req.Method,
					URL:         harURL(req.URL, cfg),
					HTTPVersion: httpVersion(req.Proto),
					Cookies:     []Cookie{},
					Headers:     harHeaders(req.Header, cfg),
					QueryString: harQueryString(req.URL.Query(), cfg),
					HeadersSize: -1,
					BodySize:    -1,
				},
			}

			waitStart := time.Now()
			resp, err := next.RoundTrip(req)
			waitMs := float64(time.Since(waitStart).Microseconds()) / 1000.0

			entry.Timings = Timings{Wait: waitMs}
			entry.Time = waitMs
			entry.Response = Response{
				Cookies:     []Cookie{},
				Headers:     []Header{},
				Content:     Content{Size: -1},
				HeadersSize: -1,
				BodySize:    -1,
			}
			if resp != nil {
				entry.Response.Status = resp.StatusCode
				entry.Response.StatusText = resp.Status
				entry.Response.HTTPVersion = httpVersion(resp.Proto)
				entry.Response.Headers = harHeaders(resp.Header, cfg)
			}

			handler(entry)
			return resp, err
		})
	}
}
