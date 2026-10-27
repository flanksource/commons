package timing

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	rpchttp "github.com/flanksource/clicky/rpc/http"
	"github.com/flanksource/commons/http/middlewares"
)

// ServerTiming records outgoing exchanges on the request's timing collector.
// The http metric ends at response headers; http_body ends at EOF or Close.
func ServerTiming(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		panic("HTTP timing requires a transport")
	}
	return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if _, ok := rpchttp.TimingsFromContext(req.Context()); !ok {
			return next.RoundTrip(req)
		}
		started := time.Now()
		resp, err := next.RoundTrip(req.WithContext(WithClientTimings(req.Context())))
		metric := rpchttp.TimingMetric{Name: "http", Duration: time.Since(started), Counters: []rpchttp.TimingCounter{{Name: "requests", Value: 1}, {Name: "errors"}, {Name: "status_errors"}}}
		if err != nil {
			metric.Counters[1].Value = 1
		}
		if resp != nil && resp.StatusCode >= http.StatusBadRequest {
			metric.Counters[2].Value = 1
		}
		rpchttp.AddTiming(req.Context(), metric)
		if resp != nil && resp.Body != nil {
			resp.Body = &timingBody{ReadCloser: resp.Body, ctx: req.Context(), started: time.Now()}
		}
		return resp, err
	})
}

type timingBody struct {
	io.ReadCloser
	ctx     context.Context
	started time.Time
	bytes   atomic.Int64
	once    sync.Once
}

func (b *timingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.bytes.Add(int64(n))
	if err != nil {
		b.finish(err)
	}
	return n, err
}

func (b *timingBody) Close() error {
	err := b.ReadCloser.Close()
	b.finish(err)
	return err
}

func (b *timingBody) finish(err error) {
	b.once.Do(func() {
		metric := rpchttp.TimingMetric{Name: "http_body", Duration: time.Since(b.started), Counters: []rpchttp.TimingCounter{{Name: "responses", Value: 1}, {Name: "bytes", Value: b.bytes.Load()}, {Name: "errors"}}}
		if err != nil && !errors.Is(err, io.EOF) {
			metric.Counters[2].Value = 1
		}
		rpchttp.AddTiming(b.ctx, metric)
	})
}
