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

func TestStripSecretsExtraKeys(t *testing.T) {
	const sessionID = "ABCDEF0123456789"

	testCases := []struct {
		name      string
		input     string
		extraKeys []string
		redacted  bool
	}{
		{
			name:     "unknown key stays visible without extra keys",
			input:    `{"session_id":"` + sessionID + `","name":"alice"}`,
			redacted: false,
		},
		{
			name:      "json field matched by an extra key",
			input:     `{"session_id":"` + sessionID + `","name":"alice"}`,
			extraKeys: []string{"session_id"},
			redacted:  true,
		},
		{
			name:      "form field matched by an extra key",
			input:     "session_id=" + sessionID + "&name=alice",
			extraKeys: []string{"session_id"},
			redacted:  true,
		},
		{
			name:      "extra key matched case-insensitively",
			input:     "Session-ID: " + sessionID,
			extraKeys: []string{"session_id"},
			redacted:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripSecrets(tc.input, tc.extraKeys...)
			if strings.Contains(got, sessionID) == tc.redacted {
				t.Errorf("StripSecrets(%q, %v) = %q, want secret redacted=%v", tc.input, tc.extraKeys, got, tc.redacted)
			}
			if !strings.Contains(got, "alice") && strings.Contains(tc.input, "alice") {
				t.Errorf("StripSecrets(%q, %v) = %q, want non-sensitive fields preserved", tc.input, tc.extraKeys, got)
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
		// A principal's name identifies who acted; it does not authenticate
		// them. Matching "user" as a substring redacted every one of these,
		// which removes the most useful half of a diagnostic and protects
		// nothing — the credential beside them is still caught.
		{"user", false},
		{"userId", false},
		{"currentUser", false},
		{"user_name", false},
		{"identity", false},
		{"username", true},
		{"UserName", true},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			if got := IsSensitiveKey(tc.key); got != tc.expected {
				t.Errorf("IsSensitiveKey(%q) = %v, want %v", tc.key, got, tc.expected)
			}
		})
	}
}

// TestMarkNonSensitive covers the escape hatch for a field whose name collides
// with a secret substring but whose value is not one. Without it the only fix
// is to rename the field, which distorts an API to satisfy a log filter.
func TestMarkNonSensitive(t *testing.T) {
	original := NonSensitiveKeys
	t.Cleanup(func() { NonSensitiveKeys = original })

	const key = "token_count"
	if !IsSensitiveKey(key) || !IsSensitiveLogKey(key) {
		t.Fatalf("precondition: %q must be redacted before it is exempted", key)
	}

	MarkNonSensitive("Token_Count")

	if IsSensitiveKey(key) {
		t.Errorf("IsSensitiveKey(%q) = true, want false after exemption", key)
	}
	// Both predicates honour the same list, so an exemption cannot hold on one
	// surface and not the other.
	if IsSensitiveLogKey(key) {
		t.Errorf("IsSensitiveLogKey(%q) = true, want false after exemption", key)
	}
	// Exact, not substring: exempting one key must not exempt everything that
	// contains it.
	if !IsSensitiveKey("token_count_secret") {
		t.Error(`IsSensitiveKey("token_count_secret") = false, want true — an exemption must not widen`)
	}

	MarkNonSensitive("token_count")
	if len(NonSensitiveKeys) != len(original)+1 {
		t.Errorf("NonSensitiveKeys grew to %d entries, want %d — re-marking must be a no-op",
			len(NonSensitiveKeys), len(original)+1)
	}
}
