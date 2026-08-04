package har_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
)

// newRegistry returns a registry whose log output is captured, so tests can
// assert on the capture announcement and none of them print to the test log.
func newRegistry(t *testing.T) (*har.Registry, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return har.NewRegistry(logger.NewWithWriter(&out)), &out
}

// setProperty sets a -P property for the duration of the test. Properties are
// process-global, so every test that touches one must restore it.
func setProperty(t *testing.T, key, value string) {
	t.Helper()
	properties.Set(key, value)
	t.Cleanup(func() { properties.Set(key, "") })
}

// jsonServer answers every request with a fixed JSON body and echoes nothing,
// so assertions are about what the registry captured, not about the server.
func jsonServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// get issues one request through transport and drains the response.
func get(t *testing.T, transport http.RoundTripper, url string, headers map[string]string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func readHAR(t *testing.T, path string) har.File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file har.File
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("HAR at %s is not valid JSON: %v", path, err)
	}
	if file.Log.Version != "1.2" {
		t.Errorf("expected HAR version 1.2, got %q", file.Log.Version)
	}
	return file
}

func TestRegistry_DisabledWithoutProperty(t *testing.T) {
	registry, _ := newRegistry(t)

	collector, path, level, err := registry.For("http")
	if err != nil {
		t.Fatal(err)
	}
	if collector != nil || path != "" || level != har.Disabled {
		t.Fatalf("expected no capture, got collector=%v path=%q level=%v", collector != nil, path, level)
	}

	base := http.DefaultTransport
	wrapped, err := registry.Transport("http", base)
	if err != nil {
		t.Fatal(err)
	}
	if wrapped != base {
		t.Error("transport must be returned unchanged when http.har is unset")
	}
}

func TestRegistry_FeaturePropertyOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	setProperty(t, "http.har", filepath.Join(dir, "global.har"))
	setProperty(t, "http.chat.har", filepath.Join(dir, "chat.har"))
	registry, _ := newRegistry(t)

	for _, tc := range []struct{ feature, want string }{
		{feature: "chat", want: "chat.har"},
		{feature: "models", want: "global.har"},
		{feature: "", want: "global.har"},
	} {
		_, path, _, err := registry.For(tc.feature)
		if err != nil {
			t.Fatal(err)
		}
		if path != filepath.Join(dir, tc.want) {
			t.Errorf("feature %q resolved to %s, want %s", tc.feature, path, tc.want)
		}
	}
}

func TestRegistry_DedupesCollectorsByAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	setProperty(t, "http.har", filepath.Join(dir, "shared.har"))
	registry, _ := newRegistry(t)

	first, _, _, err := registry.For("chat")
	if err != nil {
		t.Fatal(err)
	}
	second, _, _, err := registry.For("models")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("features writing to the same file must share one collector")
	}
}

func TestRegistry_Level(t *testing.T) {
	dir := t.TempDir()
	setProperty(t, "http.har", filepath.Join(dir, "trace.har"))

	for _, tc := range []struct {
		name    string
		value   string
		want    har.Level
		wantErr bool
	}{
		{name: "defaults to full", value: "", want: har.Full},
		{name: "metadata downgrades", value: "metadata", want: har.Metadata},
		{name: "debug is a metadata synonym", value: "debug", want: har.Metadata},
		{name: "trace is a full synonym", value: "trace", want: har.Full},
		{name: "off disables capture", value: "off", want: har.Disabled},
		{name: "a typo is an error", value: "verbose", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setProperty(t, "http.har.level", tc.value)

			registry, _ := newRegistry(t)
			_, _, level, err := registry.For("http")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for level %q", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if level != tc.want {
				t.Errorf("level = %v, want %v", level, tc.want)
			}
		})
	}
}

// TestRegistry_AnnouncesCaptureOnce pins the signal that capture is on: without
// it, a run gives no indication until the archive is flushed at exit.
func TestRegistry_AnnouncesCaptureOnce(t *testing.T) {
	const announcement = "capturing HAR to "
	dir := t.TempDir()

	t.Run("silent when capture is off", func(t *testing.T) {
		registry, out := newRegistry(t)
		if _, _, _, err := registry.For("http"); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), announcement) {
			t.Errorf("nothing should be announced when http.har is unset, got:\n%s", out)
		}
	})

	for _, tc := range []struct {
		name      string
		level     string
		sensitive string
		wantMode  string
	}{
		{name: "full is the default mode", wantMode: "level=full"},
		{name: "metadata is named", level: "metadata", wantMode: "level=metadata"},
		{name: "sensitive is called out", sensitive: "true", wantMode: "level=full, sensitive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".har")
			setProperty(t, "http.har", path)
			setProperty(t, "http.har.level", tc.level)
			setProperty(t, "http.har.sensitive", tc.sensitive)
			registry, out := newRegistry(t)

			// Two resolutions of the same archive, through both entry points.
			if _, _, _, err := registry.For("chat"); err != nil {
				t.Fatal(err)
			}
			if _, err := registry.Transport("models", http.DefaultTransport); err != nil {
				t.Fatal(err)
			}

			if got := strings.Count(out.String(), announcement); got != 1 {
				t.Fatalf("announced %d times, want exactly 1:\n%s", got, out)
			}
			want := fmt.Sprintf("%s%s (%s)", announcement, path, tc.wantMode)
			if !strings.Contains(out.String(), want) {
				t.Errorf("expected %q, got:\n%s", want, out)
			}
		})
	}

	t.Run("one line per archive", func(t *testing.T) {
		setProperty(t, "http.har", filepath.Join(dir, "global.har"))
		setProperty(t, "http.chat.har", filepath.Join(dir, "chat.har"))
		registry, out := newRegistry(t)

		for _, feature := range []string{"chat", "models"} {
			if _, _, _, err := registry.For(feature); err != nil {
				t.Fatal(err)
			}
		}
		if got := strings.Count(out.String(), announcement); got != 2 {
			t.Errorf("two distinct archives must announce twice, got %d:\n%s", got, out)
		}
	})
}

func TestRegistry_CapturesFullBodiesAndFlushes(t *testing.T) {
	const body = `{"status":"ok"}`
	path := filepath.Join(t.TempDir(), "trace.har")
	setProperty(t, "http.har", path)
	srv := jsonServer(t, body)
	registry, _ := newRegistry(t)

	transport, err := registry.Transport("http", http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	get(t, transport, srv.URL+"/ping", nil)
	get(t, transport, srv.URL+"/pong", nil)

	if err := registry.Flush(); err != nil {
		t.Fatal(err)
	}

	entries := readHAR(t, path).Log.Entries
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Request.URL, "/ping") {
		t.Errorf("unexpected first URL: %s", entries[0].Request.URL)
	}
	if entries[0].Response.Content.Text != body {
		t.Errorf("expected the response body to be captured, got %q", entries[0].Response.Content.Text)
	}

	if mode := fileMode(t, path); mode != 0o644 {
		t.Errorf("expected mode 0644 for a redacted HAR, got %04o", mode)
	}
}

func TestRegistry_MetadataLevelOmitsBodies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.har")
	setProperty(t, "http.har", path)
	setProperty(t, "http.har.level", "metadata")
	srv := jsonServer(t, `{"status":"ok"}`)
	registry, _ := newRegistry(t)

	transport, err := registry.Transport("http", http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	get(t, transport, srv.URL+"/ping", nil)
	if err := registry.Flush(); err != nil {
		t.Fatal(err)
	}

	entries := readHAR(t, path).Log.Entries
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Response.Content.Text != "" {
		t.Errorf("metadata level must not capture bodies, got %q", entries[0].Response.Content.Text)
	}
	if entries[0].Response.BodySize != -1 || entries[0].Request.BodySize != -1 {
		t.Errorf("metadata level must report unknown body sizes, got req=%d resp=%d",
			entries[0].Request.BodySize, entries[0].Response.BodySize)
	}
	if entries[0].Response.Status != http.StatusOK {
		t.Errorf("metadata level must still record the status, got %d", entries[0].Response.Status)
	}
}

func TestRegistry_SensitiveCapturesCredentialsAndTightensMode(t *testing.T) {
	const token = "Bearer sk-live-abcdefghijklmnop"
	dir := t.TempDir()
	srv := jsonServer(t, `{"status":"ok"}`)

	for _, tc := range []struct {
		name      string
		sensitive string
		wantToken bool
		wantMode  os.FileMode
	}{
		{name: "redacted by default", sensitive: "", wantMode: 0o644},
		{name: "verbatim when enabled", sensitive: "true", wantToken: true, wantMode: 0o600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".har")
			setProperty(t, "http.har", path)
			setProperty(t, "http.har.sensitive", tc.sensitive)

			registry, _ := newRegistry(t)
			transport, err := registry.Transport("http", http.DefaultTransport)
			if err != nil {
				t.Fatal(err)
			}
			get(t, transport, srv.URL+"/ping", map[string]string{"Authorization": token})
			if err := registry.Flush(); err != nil {
				t.Fatal(err)
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(raw), token); got != tc.wantToken {
				t.Errorf("token present = %v, want %v", got, tc.wantToken)
			}
			if mode := fileMode(t, path); mode != tc.wantMode {
				t.Errorf("mode = %04o, want %04o", mode, tc.wantMode)
			}
		})
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
