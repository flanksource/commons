package logger

import (
	"bytes"
	"net/http"
	"strings"
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
			name: "Redact session-id headers",
			headers: http.Header{
				"Jsessionid":  []string{"ABCDEF0123456789"},
				"X-Sessionid": []string{"ABCDEF0123456789"},
			},
			expected: http.Header{
				"Jsessionid":  []string{PrintableSecret("ABCDEF0123456789")},
				"X-Sessionid": []string{PrintableSecret("ABCDEF0123456789")},
			},
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
		{strings.Repeat("a", 65), "****,length=65"},
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

func TestStripSecrets(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "authorization header",
			input:    "Authorization: Bearer secret-token",
			expected: "Authorization: Bearer ****n",
		},
		{
			name:     "inline password assignment",
			input:    "connecting password=supersecret",
			expected: "connecting password=****t",
		},
		{
			name:     "json secret field",
			input:    `curl --data '{"password":"secret","scope":"read"}'`,
			expected: `curl --data '{"password":"s****","scope":"read"}'`,
		},
		{
			name:     "multi field secret line",
			input:    "token: abc123, refresh: true",
			expected: "token: a****, refresh: true",
		},
		{
			name:     "avoid substring false positives",
			input:    "passenger=2 keyword=foo bypass=true user_count=5",
			expected: "passenger=2 keyword=foo bypass=true user_count=5",
		},
		{
			name:     "url password and query token",
			input:    "https://user:pass@example.com/path?token=abcdef&user_count=5",
			expected: "https://user:xxxxx@example.com/path?token=a%2A%2A%2A%2A&user_count=5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripSecrets(tc.input); got != tc.expected {
				t.Errorf("StripSecrets(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestTracefRedactsSecrets(t *testing.T) {
	originalOutput := GetOutput()
	originalLevel := currentLogger.GetLevel()
	t.Cleanup(func() {
		SetOutput(originalOutput)
		currentLogger.SetLogLevel(originalLevel)
	})

	var buf bytes.Buffer
	SetOutput(&buf)
	currentLogger.SetLogLevel(Trace)

	Tracef("Authorization: %s", "Bearer secret-token")
	Tracef("%s", "download https://example.com/file%2Fname")

	got := buf.String()
	if strings.Contains(got, "secret-token") {
		t.Fatalf("Tracef leaked secret: %q", got)
	}
	if !strings.Contains(got, "Authorization: Bearer ****n") {
		t.Fatalf("Tracef did not log redacted authorization header: %q", got)
	}
	if !strings.Contains(got, "https://example.com/file%2Fname") {
		t.Fatalf("Tracef corrupted percent-encoded message: %q", got)
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
		{"JSESSIONID", true},
		{"jsessionid", true},
		{"PHPSESSID", true},
		{"ASP.NET_SessionId", true},
		{"x-sessionid", true},
		{"sessionStartTime", false},
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
