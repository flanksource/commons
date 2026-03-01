package har_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	commonshttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/logger"
)

func captureOne(t *testing.T, cfg har.HARConfig, srv *httptest.Server, method, path string, body io.Reader, reqHeaders map[string]string) *har.Entry {
	t.Helper()
	var got *har.Entry
	client := commonshttp.NewClient().HARWithConfig(cfg, func(e *har.Entry) { got = e })

	req, err := http.NewRequest(method, srv.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got == nil {
		t.Fatal("HAR handler was not called")
	}
	return got
}

func TestHAR_BasicCapture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodGet, "/ping", nil, nil)

	if entry.Request.Method != http.MethodGet {
		t.Errorf("expected method GET, got %s", entry.Request.Method)
	}
	if entry.Response.Status != 200 {
		t.Errorf("expected status 200, got %d", entry.Response.Status)
	}
	if entry.Response.Content.Text != `{"status":"ok"}` {
		t.Errorf("unexpected body: %q", entry.Response.Content.Text)
	}
	if entry.Time < 0 {
		t.Errorf("expected non-negative timing, got %f", entry.Time)
	}
	if entry.StartedDateTime == "" {
		t.Error("StartedDateTime must be set")
	}
}

func TestHAR_AuthorizationHeaderRedacted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	secret := "Bearer supersecret"
	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodGet, "/", nil, map[string]string{
		"Authorization": secret,
	})

	expected := logger.PrintableSecret(secret)
	for _, h := range entry.Request.Headers {
		if strings.EqualFold(h.Name, "Authorization") {
			if h.Value == secret {
				t.Errorf("Authorization header was not redacted, got %q", h.Value)
			}
			if h.Value != expected {
				t.Errorf("Authorization header = %q, want PrintableSecret format %q", h.Value, expected)
			}
		}
	}
}

func TestHAR_CookieHeaderRedacted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc123; Path=/")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodGet, "/", nil, map[string]string{
		"Cookie": "session=abc123",
	})

	for _, h := range entry.Request.Headers {
		if strings.EqualFold(h.Name, "Cookie") && h.Value == "session=abc123" {
			t.Errorf("Cookie header was not redacted, got %q", h.Value)
		}
	}
}

func TestHAR_BodyTruncation(t *testing.T) {
	const bodySize = 100_000
	bigBody := strings.Repeat("x", bodySize)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, bigBody)
	}))
	defer srv.Close()

	cfg := har.DefaultConfig()
	cfg.MaxBodySize = 65536
	entry := captureOne(t, cfg, srv, http.MethodGet, "/", nil, nil)

	if !entry.Response.Content.Truncated {
		t.Error("expected Content.Truncated to be true")
	}
	if int64(len(entry.Response.Content.Text)) != cfg.MaxBodySize {
		t.Errorf("expected truncated text length %d, got %d", cfg.MaxBodySize, len(entry.Response.Content.Text))
	}
	if entry.Response.Content.Size != bodySize {
		t.Errorf("expected total size %d, got %d", bodySize, entry.Response.Content.Size)
	}
}

func TestHAR_NonCapturedContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		fmt.Fprint(w, "\x89PNG\r\n\x1a\n")
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodGet, "/image.png", nil, nil)

	if entry.Response.Content.Text != "" {
		t.Errorf("expected empty body for image/png, got %q", entry.Response.Content.Text)
	}
}

func TestHAR_JSONBodyFieldRedaction(t *testing.T) {
	const password = "s3cr3tpassword"
	jsonBody := `{"username":"alice","password":"` + password + `"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodPost, "/login",
		strings.NewReader(jsonBody),
		map[string]string{"Content-Type": "application/json"})

	if entry.Request.PostData == nil {
		t.Fatal("expected PostData to be set")
	}
	if strings.Contains(entry.Request.PostData.Text, password) {
		t.Errorf("password was not redacted in request body: %s", entry.Request.PostData.Text)
	}
	if !strings.Contains(entry.Request.PostData.Text, logger.PrintableSecret(password)) {
		t.Errorf("expected printable-secret placeholder in body, got: %s", entry.Request.PostData.Text)
	}
}

func TestHAR_FormBodyFieldRedaction(t *testing.T) {
	const password = "s3cr3tpassword"
	formBody := "username=alice&password=" + password

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodPost, "/login",
		strings.NewReader(formBody),
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"})

	if entry.Request.PostData == nil {
		t.Fatal("expected PostData to be set")
	}
	if strings.Contains(entry.Request.PostData.Text, password) {
		t.Errorf("password was not redacted in form body: %s", entry.Request.PostData.Text)
	}
}

func TestHAR_NilHandlerIsNoOp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Should not panic
	client := commonshttp.NewClient().HAR(nil)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestHAR_TimingsNonNegative(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv, http.MethodGet, "/", nil, nil)

	if entry.Time < 0 {
		t.Errorf("entry.Time should be >= 0, got %f", entry.Time)
	}
	if entry.Timings.Wait < 0 {
		t.Errorf("Timings.Wait should be >= 0, got %f", entry.Timings.Wait)
	}
}
