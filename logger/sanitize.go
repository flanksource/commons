package logger

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/samber/lo"
)

var CommonRedactedHeaders = []string{
	"Authorization*", "Bearer*", "Session*", "*Cookie", "*Token", "*-Secret", "*-Key",
	"Password", "Passwd", "Pwd",
}

var SensitiveKeys = []string{"user", "pass", "key", "token", "username", "password", "authorization"}

var NonSensitiveKeys = []string{"token_type", "grant_type"}

func IsSensitiveKey(v string) bool {
	v = strings.Trim(strings.TrimSpace(strings.ToLower(v)), "_")
	for _, k := range NonSensitiveKeys {
		if v == k {
			return false
		}
	}
	for _, k := range SensitiveKeys {
		if v == k || strings.Contains(v, k) {
			return true
		}
	}
	return false
}

func SanitizeHeaders(headers http.Header, redactedHeaders ...string) http.Header {
	var redacted = http.Header{}

	redactedHeaders = append(redactedHeaders, CommonRedactedHeaders...)

	for key, values := range headers {
		key = http.CanonicalHeaderKey(key)

		if collections.MatchItems(http.CanonicalHeaderKey(key), redactedHeaders...) {
			redacted.Add(key, PrintableSecret(strings.Join(values, ",")))
			continue
		}

		redacted[key] = values
	}

	return redacted
}

// PrintableSecret returns an approximation of a secret for debugging.
// Handles structured formats:
//   - "Basic dXNlcjpwYXNz" → "Basic dXN****c"
//   - "Bearer tok_abc123"  → "Bearer tok****3"
//   - "user:password"      → "u****:p****"
//   - plain strings        → length-based redaction
func PrintableSecret(secret string) string {
	if len(secret) == 0 {
		return ""
	}

	// "Basic <cred>" or "Bearer <token>" — redact only the credential part
	if scheme, cred, ok := strings.Cut(secret, " "); ok {
		lower := strings.ToLower(scheme)
		if lower == "basic" || lower == "bearer" || lower == "token" {
			return scheme + " " + printableValue(cred)
		}
	}

	// "user:password" — redact each half independently
	if user, pass, ok := strings.Cut(secret, ":"); ok && !strings.Contains(secret, " ") {
		return printableValue(user) + ":" + printableValue(pass)
	}

	return printableValue(secret)
}

func printableValue(s string) string {
	switch {
	case len(s) == 0:
		return ""
	case len(s) > 64:
		sum := md5.Sum([]byte(s))
		hash := hex.EncodeToString(sum[:])
		return fmt.Sprintf("md5(%s),length=%d", hash[0:8], len(s))
	case len(s) > 32:
		return fmt.Sprintf("%s****%s", s[0:3], s[len(s)-1:])
	case len(s) >= 16:
		return fmt.Sprintf("%s****%s", s[0:1], s[len(s)-2:])
	case len(s) > 10:
		return fmt.Sprintf("****%s", s[len(s)-1:])
	case len(s) > 1:
		return fmt.Sprintf("%s****", s[0:1])
	default:
		return "****"
	}
}

func StripSecretsFromMap[V comparable](m map[string]V) map[string]any {
	clone := make(map[string]any)
	for k, v := range m {
		if lo.IsEmpty(v) {
			continue
		}
		if nestedMap, ok := any(v).(map[string]any); ok {
			clone[k] = StripSecretsFromMap(nestedMap)
		} else {
			if IsSensitiveKey(k) {
				clone[k] = PrintableSecret(fmt.Sprintf("%v", v))
			} else {
				clone[k] = v
			}
		}
	}
	return clone
}

// StripSecrets takes a URL, YAML or INI formatted text and removes any potentially secret data
// as denoted by keys containing "pass" or "secret" or exact matches for "key"
// the last character of the secret is kept to aid in troubleshooting
func StripSecrets(text string) string {
	if uri, err := url.Parse(text); err == nil {
		return uri.Redacted()
	}

	out := ""
	for _, line := range strings.Split(text, "\n") {

		var k, v, sep string
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			k = parts[0]
			if len(parts) > 1 {
				v = parts[1]
			}
			sep = ":"
		} else if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			k = parts[0]
			if len(parts) > 1 {
				v = parts[1]
			}
			sep = "="
		} else {
			v = line
		}

		if IsSensitiveKey(k) {
			if len(v) == 0 {
				out += k + sep + "\n"
			} else {
				out += k + sep + "****" + v[len(v)-1:] + "\n"
			}
		} else {
			out += k + sep + v + "\n"
		}
	}
	return out

}
