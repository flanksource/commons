package logger

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
				"Authorization": []string{PrintableSecret("Bearer secret-token")},
				"Cookie":        []string{PrintableSecret("session=abc123")},
				"Set-Cookie":    []string{PrintableSecret("session=abc123")},
				"Api-Key":       []string{PrintableSecret("abc123")},
				"Secret-Key":    []string{PrintableSecret("abc123")},
				"Password":      []string{PrintableSecret("abc123")},
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
				http.CanonicalHeaderKey("X-Flanksource-Access-Token"): []string{PrintableSecret("token-1")},
				http.CanonicalHeaderKey("X-Flanksource-Secret-key"):   []string{PrintableSecret("secret-1")},
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

func TestPrintableSecret(t *testing.T) {
	testCases := []struct {
		input, expected string
	}{
		{"", ""},
		{"Bearer _0XBabcdefghij1234567890abcde0", "Bearer _****e0"},
		{"Basic d2VzdG9wOnMzY3IzdA==", "Basic d****=="},
		{"alice:s3cr3tpassword", "a****:****d"},
		{"user:pw", "u****:p****"},
		{"short", "s****"},
		{"abc", "a****"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			if got := PrintableSecret(tc.input); got != tc.expected {
				t.Errorf("PrintableSecret(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	testCases := []struct {
		key      string
		expected bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"password", true},
		{"token", true},
		{"token_type", false},
		{"grant_type", false},
		{"Content-Type", false},
		{"Accept", false},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			if got := IsSensitiveKey(tc.key); got != tc.expected {
				t.Errorf("IsSensitiveKey(%q) = %v, want %v", tc.key, got, tc.expected)
			}
		})
	}
}
