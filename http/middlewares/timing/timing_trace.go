package timing

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"sync"
	"time"

	rpchttp "github.com/flanksource/clicky/rpc/http"
)

// WithClientTimings observes HTTP connection phases without replacing the
// transport. Use it for clients outside ServerTiming, such as go-git.
// Existing trace hooks are preserved. Phase durations overlap.
func WithClientTimings(ctx context.Context) context.Context {
	if _, ok := rpchttp.TimingsFromContext(ctx); !ok {
		return ctx
	}
	t := &clientTimingTrace{ctx: ctx, started: time.Now(), starts: make(map[string]time.Time)}
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { t.start("dns") },
		DNSDone:           func(info httptrace.DNSDoneInfo) { t.finish("dns", "http_dns", info.Err) },
		ConnectStart:      func(network, addr string) { t.start(network + "\x00" + addr) },
		ConnectDone:       func(network, addr string, err error) { t.finish(network+"\x00"+addr, "http_connect", err) },
		TLSHandshakeStart: func() { t.start("tls") },
		TLSHandshakeDone:  func(_ tls.ConnectionState, err error) { t.finish("tls", "http_tls", err) },
		GetConn:           func(string) { t.start("conn") },
		GotConn:           t.gotConn,
		GotFirstResponseByte: func() {
			rpchttp.AddTiming(ctx, rpchttp.TimingMetric{Name: "http_ttfb", Duration: time.Since(t.started)})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			metric := rpchttp.TimingMetric{Name: "http_wire", Counters: []rpchttp.TimingCounter{{Name: "writes", Value: 1}, {Name: "errors"}}}
			if info.Err != nil {
				metric.Counters[1].Value = 1
			}
			rpchttp.AddTiming(ctx, metric)
		},
	})
}

type clientTimingTrace struct {
	ctx     context.Context
	started time.Time
	mu      sync.Mutex
	starts  map[string]time.Time
}

func (t *clientTimingTrace) start(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.starts[key] = time.Now()
}

func (t *clientTimingTrace) finish(key, metric string, err error) {
	t.mu.Lock()
	started, ok := t.starts[key]
	delete(t.starts, key)
	t.mu.Unlock()
	if !ok {
		return
	}
	value := rpchttp.TimingMetric{Name: metric, Duration: time.Since(started), Counters: []rpchttp.TimingCounter{{Name: "ops", Value: 1}, {Name: "errors"}}}
	if err != nil {
		value.Counters[1].Value = 1
	}
	rpchttp.AddTiming(t.ctx, value)
}

func (t *clientTimingTrace) gotConn(info httptrace.GotConnInfo) {
	t.finish("conn", "http_conn", nil)
	counter := "new"
	if info.Reused {
		counter = "reused"
	}
	rpchttp.AddTiming(t.ctx, rpchttp.TimingMetric{Name: "http_conn", Counters: []rpchttp.TimingCounter{{Name: counter, Value: 1}}})
}
