package middlewares

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSanitize(t *testing.T) {
	testCases := []struct {
		name     string
		headers  http.Header
		custom   []string
		expected http.Header
	}{
		{
			name: "Redact sensitive headers",
			headers: http.Header{
				"Authorization": []string{"Bearer secret-token"},
				"Cookie":        []string{"session=abc123"},
				"Set-Cookie":    []string{"session=abc123"},
				"Api-Key":       []string{"abc123"},
				"Secret-Key":    []string{"abc123"},
				"password":      []string{"abc123"},
			},
			expected: http.Header{
				"Authorization": []string{redactedString},
				"Cookie":        []string{redactedString},
				"Set-Cookie":    []string{redactedString},
				"Api-Key":       []string{redactedString},
				"Secret-Key":    []string{redactedString},
				"Password":      []string{redactedString},
			},
		},
		{
			name: "Leave non-sensitive headers intact",
			headers: http.Header{
				"Accept-Language": []string{"en-US"},
				"User-Agent":      []string{"Mozilla/5.0"},
			},
			expected: http.Header{
				"Accept-Language": []string{"en-US"},
				"User-Agent":      []string{"Mozilla/5.0"},
			},
		},
		{
			name:     "Empty headers",
			headers:  http.Header{},
			expected: http.Header{},
		},
		{
			name:   "custom sensitive headers",
			custom: []string{"X-Flanksource-*"},
			headers: http.Header{
				"X-Flanksource-Access-Token": []string{"token-1"},
				"X-Flanksource-Secret-key":   []string{"secret-1"},
			},
			expected: http.Header{
				http.CanonicalHeaderKey("X-Flanksource-Access-Token"): []string{redactedString},
				http.CanonicalHeaderKey("X-Flanksource-Secret-key"):   []string{redactedString},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SanitizeHeaders(tc.headers, tc.custom...)
			if diff := cmp.Diff(actual, tc.expected); diff != "" {
				t.Errorf("%v", diff)
			}
		})
	}
}
