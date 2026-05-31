package http

import (
	"context"
	"io"
	netHTTP "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
)

// fakeContext implements CommonsHTTPContext for the WithContext tests.
type fakeContext struct {
	log      logger.Logger
	traceCfg TraceConfig
	traceOn  bool
	harColl  *har.Collector
	harPath  string
	harLevel HARLevel
}

func (f *fakeContext) GetLogger() logger.Logger { return f.log }
func (f *fakeContext) HTTPTraceConfig(feature string) (TraceConfig, bool) {
	return f.traceCfg, f.traceOn
}
func (f *fakeContext) HARFor(feature string) (*har.Collector, string, HARLevel) {
	return f.harColl, f.harPath, f.harLevel
}

func newTestLogger(t *testing.T, level logger.LogLevel) logger.Logger {
	t.Helper()
	l := logger.New("test-" + t.Name())
	l.SetLogLevel(level)
	return l
}

func TestWithLoggerLadder(t *testing.T) {
	cases := []struct {
		name              string
		level             logger.LogLevel
		expectTrace       bool
		wantHeaders       bool
		wantBody          bool
		wantMaxBodyLength int64
	}{
		{name: "info: no trace middleware", level: logger.Info, expectTrace: false},
		{name: "trace1: headers only", level: logger.Trace1, expectTrace: true, wantHeaders: true, wantBody: false},
		{name: "trace2: headers + body", level: logger.Trace2, expectTrace: true, wantHeaders: true, wantBody: true, wantMaxBodyLength: 4096},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := newTestLogger(t, tc.level)
			c := NewClient()
			before := len(c.transportMiddlewares)
			c = c.WithLogger(l)
			after := len(c.transportMiddlewares)

			if tc.expectTrace {
				if after != before+1 {
					t.Fatalf("expected exactly one trace middleware installed; before=%d after=%d", before, after)
				}
				if c.traceMW == nil {
					t.Fatalf("expected traceMW handle to be set")
				}
				if c.traceConfig.Headers != tc.wantHeaders {
					t.Errorf("Headers: got %v want %v", c.traceConfig.Headers, tc.wantHeaders)
				}
				if c.traceConfig.Body != tc.wantBody {
					t.Errorf("Body: got %v want %v", c.traceConfig.Body, tc.wantBody)
				}
				if c.traceConfig.MaxBodyLength != tc.wantMaxBodyLength {
					t.Errorf("MaxBodyLength: got %d want %d", c.traceConfig.MaxBodyLength, tc.wantMaxBodyLength)
				}
				if !hasHeaderCaseInsensitive(c.traceConfig.RedactedHeaders, "Authorization") {
					t.Errorf("Authorization missing from RedactedHeaders: %v", c.traceConfig.RedactedHeaders)
				}
			} else {
				if after != before {
					t.Fatalf("expected no trace middleware at Info level; before=%d after=%d", before, after)
				}
				if c.traceMW != nil {
					t.Fatalf("expected traceMW to remain nil at Info level")
				}
			}
		})
	}
}

// TestWithContextMergesTraceConfig: WithLogger at Trace1 installs the
// headers-only config; WithContext then returns a body-only TraceConfig.
// After merge the single installed middleware should have both Headers
// and Body true.
func TestWithContextMergesTraceConfig(t *testing.T) {
	l := newTestLogger(t, logger.Trace1)
	ctx := &fakeContext{
		log:      l,
		traceCfg: TraceConfig{Body: true, Response: true, MaxBodyLength: 2048},
		traceOn:  true,
	}

	c := NewClient().WithContext(ctx, "takealot")

	if c.traceMW == nil {
		t.Fatalf("traceMW must be set after WithContext")
	}

	cfg := c.traceConfig
	if !cfg.Headers {
		t.Errorf("Headers must be true after merge (from WithLogger)")
	}
	if !cfg.Body {
		t.Errorf("Body must be true after merge (from WithContext)")
	}
	if !cfg.Response {
		t.Errorf("Response must be true after merge")
	}
	if cfg.MaxBodyLength != 2048 {
		t.Errorf("MaxBodyLength: got %d want 2048", cfg.MaxBodyLength)
	}
	if !hasHeaderCaseInsensitive(cfg.RedactedHeaders, "Authorization") {
		t.Errorf("Authorization must be redacted after WithContext: %v", cfg.RedactedHeaders)
	}
}

// TestTraceToStdoutDedupe: calling TraceToStdout twice on the same
// client should install one middleware whose config is the merge of the
// two inputs.
func TestTraceToStdoutDedupe(t *testing.T) {
	c := NewClient()
	before := len(c.transportMiddlewares)
	c = c.TraceToStdout(TraceConfig{Headers: true})
	afterFirst := len(c.transportMiddlewares)
	if afterFirst != before+1 {
		t.Fatalf("first TraceToStdout must install one middleware; got %d", afterFirst-before)
	}
	c = c.TraceToStdout(TraceConfig{Body: true, MaxBodyLength: 1024})
	afterSecond := len(c.transportMiddlewares)
	if afterSecond != afterFirst {
		t.Fatalf("second TraceToStdout must not stack; got %d middlewares (was %d)", afterSecond, afterFirst)
	}
	if !c.traceConfig.Headers || !c.traceConfig.Body {
		t.Fatalf("merged config must have both Headers and Body; got %+v", c.traceConfig)
	}
	if c.traceConfig.MaxBodyLength != 1024 {
		t.Errorf("MaxBodyLength: got %d want 1024", c.traceConfig.MaxBodyLength)
	}
}

// TestWithContextMetadataHAR: a metadata-level HAR collector must
// produce one entry per outbound request with bodySize == -1 (no body
// capture).
func TestWithContextMetadataHAR(t *testing.T) {
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := &fakeContext{
		log:      newTestLogger(t, logger.Info),
		harColl:  collector,
		harPath:  "/dev/null",
		harLevel: HARMetadata,
	}
	c := NewClient().WithContext(ctx, "takealot")

	resp, err := c.R(context.Background()).Post(srv.URL, strings.NewReader("hello-world"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}

	entries := collector.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 HAR entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Request.BodySize != -1 {
		t.Errorf("metadata HAR must not capture body size; got %d want -1", e.Request.BodySize)
	}
	if e.Response.Content.Size != -1 {
		t.Errorf("metadata HAR must not capture response content size; got %d want -1", e.Response.Content.Size)
	}
	if e.Request.Method != "POST" {
		t.Errorf("method: got %q want POST", e.Request.Method)
	}
}

// TestWithContextFullHAR: a full HAR collector path should produce HAR
// entries via the collector's body-capturing middleware.
func TestWithContextFullHAR(t *testing.T) {
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok-full"))
	}))
	defer srv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	ctx := &fakeContext{
		log:      newTestLogger(t, logger.Info),
		harColl:  collector,
		harPath:  "/dev/null",
		harLevel: HARFull,
	}
	c := NewClient().WithContext(ctx, "takealot")

	if c.harPath != "/dev/null" {
		t.Errorf("harPath: got %q want /dev/null", c.harPath)
	}

	resp, err := c.R(context.Background()).Post(srv.URL, strings.NewReader("hello-world"))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}

	entries := collector.Entries()
	if len(entries) == 0 {
		t.Fatalf("expected at least 1 HAR entry under HARFull")
	}
}

func hasHeaderCaseInsensitive(headers []string, want string) bool {
	for _, h := range headers {
		if strings.EqualFold(h, want) {
			return true
		}
	}
	return false
}
