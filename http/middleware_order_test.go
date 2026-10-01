package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"slices"
	"testing"

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
