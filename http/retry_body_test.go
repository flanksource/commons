package http_test

import (
	"context"
	"io"
	netHTTP "net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/properties"
)

// stallingServer records every request body it receives and stalls the first
// attempt past the client timeout, forcing the transport-error retry path.
func stallingServer(t *testing.T, bodies *[]string, mu *sync.Mutex, attempts *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		*attempts++
		n := *attempts
		*bodies = append(*bodies, string(b))
		mu.Unlock()
		if n == 1 {
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(netHTTP.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// A retried POST must replay the request body on every attempt. roundTrip
// drains the body reader, so before the fix a retry sent an empty body — the
// downstream JSON validation failure. The retry path fires on a transport
// error, so the server stalls the first attempt past the client timeout to
// force it, then records the body every attempt received.
func TestRetryReplaysRequestBody(t *testing.T) {
	const payload = `{"hello":"world"}`

	var (
		mu       sync.Mutex
		bodies   []string
		attempts int
	)
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		attempts++
		n := attempts
		bodies = append(bodies, string(b))
		mu.Unlock()
		if n == 1 {
			// Stall past the client timeout to trigger the transport-error
			// retry path (the production processDynamicFieldChange timeout).
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(netHTTP.StatusOK)
	}))
	defer srv.Close()

	resp, err := http.NewClient().
		BaseURL(srv.URL).
		Timeout(50*time.Millisecond).
		Retry(3, time.Millisecond, 1.0).
		R(context.Background()).
		Post("/", payload)
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if !resp.IsOK() {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts < 2 {
		t.Fatalf("expected a retry (>=2 attempts), got %d", attempts)
	}
	for i, got := range bodies {
		if got != payload {
			t.Errorf("attempt %d received body %q, want %q (body must replay on retry)", i+1, got, payload)
		}
	}
}

// The same guarantee holds when the body is supplied as a raw io.Reader: it is
// buffered once so the retry can replay it rather than re-reading an exhausted
// stream.
func TestRetryReplaysReaderBody(t *testing.T) {
	const payload = "raw-reader-body"

	var (
		mu       sync.Mutex
		bodies   []string
		attempts int
	)
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, r *netHTTP.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		attempts++
		n := attempts
		bodies = append(bodies, string(b))
		mu.Unlock()
		if n == 1 {
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(netHTTP.StatusOK)
	}))
	defer srv.Close()

	resp, err := http.NewClient().
		BaseURL(srv.URL).
		Timeout(50*time.Millisecond).
		Retry(3, time.Millisecond, 1.0).
		R(context.Background()).
		Post("/", io.Reader(strings.NewReader(payload)))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if !resp.IsOK() {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	for i, got := range bodies {
		if got != payload {
			t.Errorf("attempt %d received body %q, want %q (reader body must buffer and replay)", i+1, got, payload)
		}
	}
}

// A reader body at/under the configured cap is still buffered and replayed on
// retry, exactly as the default case — the cap only changes behavior above it.
func TestRequestBodyUnderCapStillBuffers(t *testing.T) {
	properties.Set(http.MaxBufferSizeProperty, 1024)
	t.Cleanup(func() { properties.Set(http.MaxBufferSizeProperty, "") })

	payload := strings.Repeat("a", 512) // under the 1024 cap

	var (
		mu       sync.Mutex
		bodies   []string
		attempts int
	)
	srv := stallingServer(t, &bodies, &mu, &attempts)

	resp, err := http.NewClient().
		BaseURL(srv.URL).
		Timeout(50*time.Millisecond).
		Retry(3, time.Millisecond, 1.0).
		R(context.Background()).
		Post("/", io.Reader(strings.NewReader(payload)))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if !resp.IsOK() {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts < 2 {
		t.Fatalf("expected a retry (>=2 attempts), got %d", attempts)
	}
	for i, got := range bodies {
		if got != payload {
			t.Errorf("attempt %d received body %q (len %d), want full payload (len %d): body under cap must replay", i+1, got, len(got), len(payload))
		}
	}
}

// A reader body over the cap streams through un-buffered: the first attempt
// receives the full payload, but a retry is refused with an explicit error
// rather than silently resending an empty body.
func TestRequestBodyOverCapStreamsOnceAndRefusesRetry(t *testing.T) {
	properties.Set(http.MaxBufferSizeProperty, 16)
	t.Cleanup(func() { properties.Set(http.MaxBufferSizeProperty, "") })

	payload := strings.Repeat("b", 1024) // well over the 16-byte cap

	var (
		mu       sync.Mutex
		bodies   []string
		attempts int
	)
	srv := stallingServer(t, &bodies, &mu, &attempts)

	_, err := http.NewClient().
		BaseURL(srv.URL).
		Timeout(50*time.Millisecond).
		Retry(3, time.Millisecond, 1.0).
		R(context.Background()).
		Post("/", io.Reader(strings.NewReader(payload)))
	if err == nil {
		t.Fatal("expected retry of an over-cap streamed body to error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot retry request: body exceeded") {
		t.Fatalf("expected over-cap retry-refusal error, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) == 0 {
		t.Fatal("server received no request")
	}
	if bodies[0] != payload {
		t.Errorf("first attempt received body len %d, want full payload len %d: over-cap body must stream intact once", len(bodies[0]), len(payload))
	}
}

// Setting the cap to 0 disables it: a body larger than the default 4 MB cap is
// fully buffered and replayed on retry.
func TestRequestBodyCapDisabled(t *testing.T) {
	properties.Set(http.MaxBufferSizeProperty, 0)
	t.Cleanup(func() { properties.Set(http.MaxBufferSizeProperty, "") })

	payload := strings.Repeat("c", 5*1024*1024) // > 4 MB default cap

	var (
		mu       sync.Mutex
		bodies   []string
		attempts int
	)
	srv := stallingServer(t, &bodies, &mu, &attempts)

	resp, err := http.NewClient().
		BaseURL(srv.URL).
		Timeout(50*time.Millisecond).
		Retry(3, time.Millisecond, 1.0).
		R(context.Background()).
		Post("/", io.Reader(strings.NewReader(payload)))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if !resp.IsOK() {
		t.Fatalf("expected eventual 200, got %d", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts < 2 {
		t.Fatalf("expected a retry (>=2 attempts), got %d", attempts)
	}
	for i, got := range bodies {
		if got != payload {
			t.Errorf("attempt %d received body len %d, want full payload len %d: disabled cap must buffer and replay", i+1, len(got), len(payload))
		}
	}
}
