package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flanksource/commons/http/middlewares"
)

// TestUsePreservesRegistrationOrder pins the builder contract: middleware runs
// in the order it was registered, whether it was registered in one Use call or
// in several chained ones.
func TestUsePreservesRegistrationOrder(t *testing.T) {
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	record := func(order *[]string, name string) middlewares.Middleware {
		return func(next stdhttp.RoundTripper) stdhttp.RoundTripper {
			return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
				*order = append(*order, name)
				return next.RoundTrip(req)
			})
		}
	}

	testCases := []struct {
		name  string
		build func(order *[]string) *Client
	}{
		{
			name: "single Use call",
			build: func(order *[]string) *Client {
				return NewClient().Use(record(order, "first"), record(order, "second"))
			},
		},
		{
			name: "chained Use calls",
			build: func(order *[]string) *Client {
				return NewClient().Use(record(order, "first")).Use(record(order, "second"))
			},
		},
	}

	expected := []string{"first", "second"}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var order []string
			response, err := tc.build(&order).R(context.Background()).Get(server.URL)
			if err != nil {
				t.Fatalf("GET %s: %v", server.URL, err)
			}
			defer response.Body.Close()

			if !slices.Equal(order, expected) {
				t.Errorf("middleware ran in order %v, want %v", order, expected)
			}
		})
	}
}

// TestTransportSettersKeepMiddleware: transport setters called after Use must
// modify the transport beneath the middleware chain instead of panicking on
// the wrapped RoundTripper or dropping the middleware.
func TestTransportSettersKeepMiddleware(t *testing.T) {
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	calls := 0
	counter := func(next stdhttp.RoundTripper) stdhttp.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			calls++
			return next.RoundTrip(req)
		})
	}

	client := NewClient().Use(counter).InsecureSkipVerify(false).DisableKeepAlive(true)
	if _, err := client.TLSConfig(TLSConfig{HandshakeTimeout: 5 * time.Second}); err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}

	base, ok := client.baseTransport.(*stdhttp.Transport)
	if !ok {
		t.Fatalf("base transport is %T, want *http.Transport", client.baseTransport)
	}
	if !base.DisableKeepAlives || base.TLSHandshakeTimeout != 5*time.Second {
		t.Errorf("base transport settings not applied: DisableKeepAlives=%v TLSHandshakeTimeout=%v", base.DisableKeepAlives, base.TLSHandshakeTimeout)
	}

	response, err := client.R(context.Background()).Get(server.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", server.URL, err)
	}
	defer response.Body.Close()
	if calls != 1 {
		t.Errorf("middleware ran %d times, want 1", calls)
	}
}

// TestTransportSettersRejectCustomTransport: transport setters cannot modify a
// base transport that is not an *http.Transport, so they report an error
// instead of panicking, and requests made through the client fail with it.
func TestTransportSettersRejectCustomTransport(t *testing.T) {
	custom := middlewares.RoundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
		t.Fatal("custom transport must not be reached")
		return nil, nil
	})

	client := NewClient()
	client.httpClient.Transport = custom
	if _, err := client.TLSConfig(TLSConfig{}); err == nil || !strings.Contains(err.Error(), "expected *http.Transport") {
		t.Errorf("TLSConfig error = %v, want an unsupported-transport error", err)
	}

	client.InsecureSkipVerify(true)
	_, err := client.R(context.Background()).Get("https://example.invalid")
	if err == nil || !strings.Contains(err.Error(), "InsecureSkipVerify") {
		t.Errorf("request error = %v, want the recorded InsecureSkipVerify transport error", err)
	}
}
