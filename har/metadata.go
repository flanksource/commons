package har

import (
	"net/http"
	"time"

	"github.com/flanksource/commons/http/middlewares"
)

// NewMetadataMiddleware captures method, URL, sanitized headers, query string,
// status and timings — no request or response bodies. Body sizes use -1 per the
// HAR spec ("size unknown"). Use it when you want a HAR file for traffic
// analysis without paying the body-buffering cost. handler receives each entry
// once the request completes; use Collector.MetadataMiddleware to also see
// requests while they are in flight.
//
// Ported from duty/connection/common.go's metadataHARMiddleware, which
// commons/http and commons-db each carried their own copy of.
func NewMetadataMiddleware(cfg HARConfig, handler func(*Entry)) middlewares.Middleware {
	return handlerMiddleware(captureMetadata, cfg, handler)
}

func captureMetadata(req *http.Request, next http.RoundTripper, cfg HARConfig, record recordFunc) (*http.Response, error) {
	started := time.Now()
	entry := &Entry{
		StartedDateTime: started.UTC().Format(startedDateTimeLayout),
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
	complete := record(started, *entry)

	resp, err := next.RoundTrip(req)

	entry.Timings = Timings{Wait: millis(time.Since(started))}
	entry.Time = entry.Timings.Wait
	entry.Response = Response{
		Cookies:     []Cookie{},
		Headers:     []Header{},
		Content:     Content{Size: -1},
		HeadersSize: -1,
		BodySize:    -1,
	}
	if resp != nil {
		entry.Response.Status = resp.StatusCode
		entry.Response.StatusText = http.StatusText(resp.StatusCode)
		entry.Response.HTTPVersion = httpVersion(resp.Proto)
		entry.Response.Headers = harHeaders(resp.Header, cfg)
	}
	if err != nil {
		entry.Error = err.Error()
	}

	complete(entry)
	return resp, err
}
