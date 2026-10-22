package http_test

import (
	"context"
	"errors"
	"io"
	netHTTP "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flanksource/commons/http"
)

const errorPayload = `{"errorMessage":"FooException: boom"}`

func payloadServer(t *testing.T, status int, payload string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(netHTTP.HandlerFunc(func(w netHTTP.ResponseWriter, _ *netHTTP.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, client *http.Client) (*http.Response, error) {
	t.Helper()
	return client.R(context.Background()).Get("/")
}

// Reading an unbuffered body a second time used to return "" with a nil error,
// which is how a decoded OIPA 500 lost its message. It must fail loudly.
func TestUnbufferedSecondReadReturnsErrBodyConsumed(t *testing.T) {
	srv := payloadServer(t, netHTTP.StatusInternalServerError, errorPayload)
	resp, err := get(t, http.NewClient().BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	var body map[string]any
	if err := resp.Into(&body); err != nil {
		t.Fatalf("first Into errored: %v", err)
	}

	raw, err := resp.AsString()
	if !errors.Is(err, http.ErrBodyConsumed) {
		t.Fatalf("AsString after Into = (%q, %v), want ErrBodyConsumed", raw, err)
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("error %q does not name the request URL %s", err, srv.URL)
	}
	if _, err := resp.AsJSON(); !errors.Is(err, http.ErrBodyConsumed) {
		t.Errorf("AsJSON after Into = %v, want ErrBodyConsumed", err)
	}
}

// A caller reading resp.Body directly consumes it just the same.
func TestUnbufferedDirectBodyReadThenAsStringErrors(t *testing.T) {
	srv := payloadServer(t, netHTTP.StatusOK, errorPayload)
	resp, err := get(t, http.NewClient().BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("direct read errored: %v", err)
	}
	if _, err := resp.AsString(); !errors.Is(err, http.ErrBodyConsumed) {
		t.Fatalf("AsString after a direct Body read = %v, want ErrBodyConsumed", err)
	}
}

func TestUnbufferedSingleReadReturnsBody(t *testing.T) {
	srv := payloadServer(t, netHTTP.StatusOK, errorPayload)
	resp, err := get(t, http.NewClient().BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	if got, err := resp.AsString(); err != nil || got != errorPayload {
		t.Fatalf("AsString = (%q, %v), want (%q, nil)", got, err, errorPayload)
	}
}

// A buffered body can be decoded and then read as text, any number of times,
// and a raw reader of resp.Body still sees the whole payload.
func TestBufferedBodyReadsRepeatedly(t *testing.T) {
	srv := payloadServer(t, netHTTP.StatusInternalServerError, errorPayload)
	resp, err := get(t, http.NewClient().BaseURL(srv.URL).BufferBody(0))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	var body map[string]any
	if err := resp.Into(&body); err != nil {
		t.Fatalf("Into errored: %v", err)
	}
	if body["errorMessage"] != "FooException: boom" {
		t.Errorf("Into decoded %v", body)
	}
	for i := range 2 {
		if got, err := resp.AsString(); err != nil || got != errorPayload {
			t.Fatalf("AsString #%d = (%q, %v), want (%q, nil)", i+1, got, err, errorPayload)
		}
	}
	if raw, err := io.ReadAll(resp.Body); err != nil || string(raw) != errorPayload {
		t.Fatalf("direct Body read = (%q, %v), want (%q, nil)", raw, err, errorPayload)
	}
}

func TestBufferedBodyLimit(t *testing.T) {
	const limit = 16
	for name, tc := range map[string]struct {
		size    int
		wantErr bool
	}{
		"exactly the limit": {size: limit},
		"one over the limit": {size: limit + 1, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			payload := strings.Repeat("x", tc.size)
			srv := payloadServer(t, netHTTP.StatusOK, payload)
			resp, err := get(t, http.NewClient().BaseURL(srv.URL).BufferBody(limit))
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "exceeds 16 byte buffer limit") {
					t.Fatalf("request = %v, want the buffer limit error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("request errored: %v", err)
			}
			if got, err := resp.AsString(); err != nil || got != payload {
				t.Fatalf("AsString = (%q, %v), want the %d byte payload", got, err, tc.size)
			}
		})
	}
}

// BufferBody(0) selects the 16MB default: a body just over 4MB (the request
// body cap) is still buffered in full.
func TestBufferBodyZeroUsesDefaultLimit(t *testing.T) {
	if http.DefaultMaxBufferedBodyBytes != 16<<20 {
		t.Fatalf("DefaultMaxBufferedBodyBytes = %d, want 16MB", http.DefaultMaxBufferedBodyBytes)
	}
	payload := strings.Repeat("y", 4<<20+1)
	srv := payloadServer(t, netHTTP.StatusOK, payload)
	resp, err := get(t, http.NewClient().BaseURL(srv.URL).BufferBody(0))
	if err != nil {
		t.Fatalf("request errored: %v", err)
	}
	for range 2 {
		if got, err := resp.AsString(); err != nil || len(got) != len(payload) {
			t.Fatalf("AsString = (len %d, %v), want len %d", len(got), err, len(payload))
		}
	}
}
