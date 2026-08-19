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
	if entry.Response.StatusText != "OK" {
		t.Errorf("expected HAR status text OK, got %q", entry.Response.StatusText)
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

// TestHAR_CaptureSensitiveKeepsCredentials pins the -Phttp.har.sensitive escape
// hatch at the config layer: with it set, the archive is replayable because
// every value is verbatim, including a query parameter the default heuristics
// would otherwise mask.
func TestHAR_CaptureSensitiveKeepsCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	const (
		secret   = "Bearer supersecret"
		apiKey   = "sk-live-1234567890"
		jsonBody = `{"api_key":"sk-live-1234567890"}`
	)
	cfg := har.DefaultConfig()
	cfg.CaptureSensitive = true

	entry := captureOne(t, cfg, srv, http.MethodPost, "/?token="+apiKey,
		strings.NewReader(jsonBody), map[string]string{
			"Authorization": secret,
			"Content-Type":  "application/json",
		})

	if got := headerValue(entry.Request.Headers, "Authorization"); got != secret {
		t.Errorf("Authorization = %q, want the verbatim value %q", got, secret)
	}
	if !strings.Contains(entry.Request.URL, apiKey) {
		t.Errorf("URL %q dropped the query credential", entry.Request.URL)
	}
	if entry.Request.PostData == nil || entry.Request.PostData.Text != jsonBody {
		t.Errorf("request body was redacted: %+v", entry.Request.PostData)
	}
	for _, q := range entry.Request.QueryString {
		if q.Name == "token" && q.Value != apiKey {
			t.Errorf("query string token = %q, want %q", q.Value, apiKey)
		}
	}
}

func headerValue(headers []har.Header, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
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
		w.Header().Set("Content-Length", fmt.Sprintf("%d", bodySize))
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

func TestHAR_QueryStringCredentialsRedacted(t *testing.T) {
	const password = "s3cr3tpassword"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	entry := captureOne(t, har.DefaultConfig(), srv,
		http.MethodGet, "/?username=alice&password="+password, nil, nil)

	for _, q := range entry.Request.QueryString {
		if strings.EqualFold(q.Name, "password") && q.Value == password {
			t.Errorf("password query param was not redacted, got %q", q.Value)
		}
	}
	if strings.Contains(entry.Request.URL, password) {
		t.Errorf("password leaked in captured URL: %s", entry.Request.URL)
	}
}

func TestHAR_RedactedBodyKeysScrubExtraIdentifiers(t *testing.T) {
	const personalID = "9001011234567"
	const session = "ABCDEF0123456789"
	// PersonalId appears both at the top level and nested inside an array of
	// objects — neither must survive redaction.
	jsonBody := `{"PersonalId":"` + personalID + `","name":"alice","members":[{"PersonalId":"` + personalID + `"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := har.DefaultConfig()
	cfg.RedactedBodyKeys = []string{"PersonalId", "jsessionid"}

	entry := captureOne(t, cfg, srv,
		http.MethodPost, "/login?jsessionid="+session,
		strings.NewReader(jsonBody),
		map[string]string{"Content-Type": "application/json"})

	if entry.Request.PostData == nil {
		t.Fatal("expected PostData to be set")
	}
	if strings.Contains(entry.Request.PostData.Text, personalID) {
		t.Errorf("PersonalId was not redacted in body: %s", entry.Request.PostData.Text)
	}
	if !strings.Contains(entry.Request.PostData.Text, "alice") {
		t.Errorf("non-sensitive field dropped from body: %s", entry.Request.PostData.Text)
	}
	for _, q := range entry.Request.QueryString {
		if strings.EqualFold(q.Name, "jsessionid") && q.Value == session {
			t.Errorf("jsessionid query param was not redacted, got %q", q.Value)
		}
	}
	if strings.Contains(entry.Request.URL, session) {
		t.Errorf("jsessionid leaked in captured URL: %s", entry.Request.URL)
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
