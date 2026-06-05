package http

import (
	"bytes"
	"context"
	"io"
	netHTTP "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
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
		wantAccessLog     bool
		wantHeaders       bool
		wantFormParams    bool
		wantTLS           bool
		wantBody          bool
		wantResponse      bool
		wantMaxBodyLength int64
	}{
		{name: "info: error-only access log installed", level: logger.Info, expectTrace: true, wantAccessLog: true},
		{name: "debug: access log only", level: logger.Debug, expectTrace: true, wantAccessLog: true},
		{name: "trace: params + headers", level: logger.Trace, expectTrace: true, wantAccessLog: true, wantHeaders: true, wantFormParams: true},
		{name: "trace1: request body + tls", level: logger.Trace1, expectTrace: true, wantAccessLog: true, wantHeaders: true, wantFormParams: true, wantTLS: true, wantBody: true, wantMaxBodyLength: 4096},
		{name: "trace2: response body", level: logger.Trace2, expectTrace: true, wantAccessLog: true, wantHeaders: true, wantFormParams: true, wantTLS: true, wantBody: true, wantResponse: true, wantMaxBodyLength: 4096},
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
				if c.traceConfig.AccessLog != tc.wantAccessLog {
					t.Errorf("AccessLog: got %v want %v", c.traceConfig.AccessLog, tc.wantAccessLog)
				}
				if c.traceConfig.FormParams != tc.wantFormParams {
					t.Errorf("FormParams: got %v want %v", c.traceConfig.FormParams, tc.wantFormParams)
				}
				if c.traceConfig.TLS != tc.wantTLS {
					t.Errorf("TLS: got %v want %v", c.traceConfig.TLS, tc.wantTLS)
				}
				if c.traceConfig.Body != tc.wantBody {
					t.Errorf("Body: got %v want %v", c.traceConfig.Body, tc.wantBody)
				}
				if c.traceConfig.Response != tc.wantResponse {
					t.Errorf("Response: got %v want %v", c.traceConfig.Response, tc.wantResponse)
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

func TestTraceConfigForLogLevelBaseLevel(t *testing.T) {
	properties.Set("http.log.base-level", "")
	t.Setenv("HTTP_LOG_BASE_LEVEL", "")

	cfg, ok := TraceConfigForLogLevel(logger.Info, WithTraceBaseLevel(logger.Info))
	if !ok {
		t.Fatalf("Info should enable trace when base is Info")
	}
	if !cfg.AccessLog || cfg.Headers || cfg.Body || cfg.Response {
		t.Fatalf("Info/base Info config = %+v, want access only", cfg)
	}

	cfg, ok = TraceConfigForLogLevel(logger.Trace, WithTraceBaseLevel(logger.Info))
	if !ok {
		t.Fatalf("Trace should enable trace when base is Info")
	}
	if !cfg.Body || !cfg.TLS || cfg.Response {
		t.Fatalf("Trace/base Info config = %+v, want request body + tls but no response", cfg)
	}

	// One level below base installs the access log in error-only mode.
	if cfg, ok := TraceConfigForLogLevel(logger.Debug, WithTraceBaseLevel(logger.Trace)); !ok {
		t.Fatalf("Debug (base-1 of a Trace base) should install the error-only access log")
	} else if !cfg.AccessLog || cfg.Headers || cfg.Body || cfg.Response {
		t.Fatalf("Debug/base Trace config = %+v, want access only", cfg)
	}

	// Two levels below base installs nothing.
	if _, ok := TraceConfigForLogLevel(logger.Info, WithTraceBaseLevel(logger.Trace)); ok {
		t.Fatalf("Info (base-2 of a Trace base) should install no trace middleware")
	}
}

func TestTraceConfigForLogLevelReadsConfiguredBase(t *testing.T) {
	properties.Set("http.log.base-level", "info")
	t.Cleanup(func() { properties.Set("http.log.base-level", "") })
	t.Setenv("HTTP_LOG_BASE_LEVEL", "")

	cfg, ok := TraceConfigForLogLevel(logger.Info)
	if !ok {
		t.Fatalf("Info should enable trace when http.log.base-level=info")
	}
	if !cfg.AccessLog || cfg.Headers || cfg.Body {
		t.Fatalf("Info/property base config = %+v, want access only", cfg)
	}
}

// TestWithContextMergesTraceConfig: WithLogger at Trace installs the
// params+headers config; WithContext then returns a body-only TraceConfig.
// After merge the single installed middleware should have Headers and Body.
func TestWithContextMergesTraceConfig(t *testing.T) {
	l := newTestLogger(t, logger.Trace)
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

func TestWithLoggerHTTPOutputLadder(t *testing.T) {
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("X-Response-Key", "response-secret")
		w.WriteHeader(netHTTP.StatusCreated)
		_, _ = w.Write([]byte("response-body"))
	}))
	defer srv.Close()

	cases := []struct {
		name    string
		level   logger.LogLevel
		want    []string
		wantNot []string
	}{
		{
			name:    "debug access line",
			level:   logger.Debug,
			want:    []string{"POST", "201"},
			wantNot: []string{"q=visible", "Query Params", "Form Params", "X-Api-Key", "foo", "response-body"},
		},
		{
			name:    "trace params and headers",
			level:   logger.Trace,
			want:    []string{"POST", "201", "Query Params", "q", "visible", "Form Params", "foo", "bar", "X-Api-Key", "X-Response-Key"},
			wantNot: []string{"rawsecret", "supersecret", "response-secret", "response-body"},
		},
		{
			name:    "trace1 request body only",
			level:   logger.Trace1,
			want:    []string{"POST", "201", "foo", "bar"},
			wantNot: []string{"supersecret", "response-body"},
		},
		{
			name:    "trace2 response body",
			level:   logger.Trace2,
			want:    []string{"POST", "201", "foo", "bar", "response-body"},
			wantNot: []string{"supersecret", "response-secret"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureLogOutput(t)
			l := newTestLogger(t, tc.level)
			resp, err := NewClient().
				WithLogger(l).
				Header("Content-Type", "application/x-www-form-urlencoded").
				Header("X-Api-Key", "rawsecret").
				R(context.Background()).
				QueryParam("q", "visible").
				Post(srv.URL+"/submit", "foo=bar&password=supersecret")
			if err != nil {
				t.Fatalf("Post: %v", err)
			}
			defer resp.Body.Close()

			got := out.String()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			for _, wantNot := range tc.wantNot {
				if strings.Contains(got, wantNot) {
					t.Errorf("output unexpectedly contains %q:\n%s", wantNot, got)
				}
			}
		})
	}
}

func TestWithLoggerRequestBodyStartsAtTrace1(t *testing.T) {
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(netHTTP.StatusOK)
	}))
	defer srv.Close()

	for _, tc := range []struct {
		name     string
		level    logger.LogLevel
		wantBody bool
	}{
		{name: "trace omits raw request body", level: logger.Trace, wantBody: false},
		{name: "trace1 prints request body", level: logger.Trace1, wantBody: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := captureLogOutput(t)
			l := newTestLogger(t, tc.level)
			resp, err := NewClient().
				WithLogger(l).
				Header("Content-Type", "application/json").
				R(context.Background()).
				Post(srv.URL+"/json", `{"message":"request-body-marker"}`)
			if err != nil {
				t.Fatalf("Post: %v", err)
			}
			defer resp.Body.Close()

			contains := strings.Contains(out.String(), "request-body-marker")
			if contains != tc.wantBody {
				t.Fatalf("request body presence = %v, want %v:\n%s", contains, tc.wantBody, out.String())
			}
		})
	}
}

// TestErrorResponseLoggedAtInfo asserts the error-only access log behaviour:
// at INFO (-v=0) a >= 400 response logs its access line and body, while a 2xx
// response stays silent; at DEBUG (-v=1) the 2xx access line appears.
func TestErrorResponseLoggedAtInfo(t *testing.T) {
	const html = "<html><body><h1>HTTP Status 404 – Not Found</h1></body></html>"
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		_, _ = io.ReadAll(r.Body)
		if strings.HasSuffix(r.URL.Path, "/missing") {
			w.WriteHeader(netHTTP.StatusNotFound)
			_, _ = w.Write([]byte(html))
			return
		}
		w.WriteHeader(netHTTP.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	t.Run("404 surfaces access line and body at INFO", func(t *testing.T) {
		out := captureLogOutput(t)
		resp, err := NewClient().WithLogger(newTestLogger(t, logger.Info)).
			R(context.Background()).Get(srv.URL + "/missing")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		defer resp.Body.Close()

		got := out.String()
		for _, want := range []string{"GET", "404", "HTTP Status 404"} {
			if !strings.Contains(got, want) {
				t.Errorf("INFO output missing %q:\n%s", want, got)
			}
		}

		// The body must still be readable downstream (readErrorBody restores it).
		body, _ := io.ReadAll(resp.Body)
		if string(body) != html {
			t.Errorf("response body not restored after logging: got %q", string(body))
		}
	})

	t.Run("2xx is silent at INFO", func(t *testing.T) {
		out := captureLogOutput(t)
		resp, err := NewClient().WithLogger(newTestLogger(t, logger.Info)).
			R(context.Background()).Get(srv.URL + "/ok")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		defer resp.Body.Close()

		if got := strings.TrimSpace(out.String()); got != "" {
			t.Errorf("a successful request must not log at INFO; got:\n%s", got)
		}
	})

	t.Run("2xx access line appears at DEBUG", func(t *testing.T) {
		out := captureLogOutput(t)
		resp, err := NewClient().WithLogger(newTestLogger(t, logger.Debug)).
			R(context.Background()).Get(srv.URL + "/ok")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		defer resp.Body.Close()

		got := out.String()
		for _, want := range []string{"GET", "200"} {
			if !strings.Contains(got, want) {
				t.Errorf("DEBUG output missing %q:\n%s", want, got)
			}
		}
	})
}

func captureLogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	prev := logger.GetOutput()
	logger.SetOutput(&out)
	t.Cleanup(func() { logger.SetOutput(prev) })
	return &out
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
