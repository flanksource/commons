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

var SensitiveKeys = []string{"user", "pass", "key", "token", "username", "password"}

const redactedString = "████████████████████"

func IsSensitiveKey(v string) bool {
	v = strings.Trim(strings.TrimSpace(strings.ToLower(v)), "_")
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
			redacted.Add(key, redactedString)
			continue
		}

		redacted[key] = values
	}

	return redacted
}

// PrintableSecret returns an approximation of a secret, so that it is possible to compare the secrets rudimentally
// e.g. for "john-doe-jane" it will return ***e
// Secrets smaller than 10 characters will always return ***
// These secrets
func PrintableSecret(secret string) string {
	if len(secret) == 0 {
		return ""
	} else if len(secret) > 64 {
		sum := md5.Sum([]byte(secret))
		hash := hex.EncodeToString(sum[:])
		return fmt.Sprintf("md5(%s),length=%d", hash[0:8], len(secret))
	} else if len(secret) > 32 {
		return fmt.Sprintf("%s****%s", secret[0:3], secret[len(secret)-1:])
	} else if len(secret) >= 16 {
		return fmt.Sprintf("%s****%s", secret[0:1], secret[len(secret)-2:])
	} else if len(secret) > 10 {
		return fmt.Sprintf("****%s", secret[len(secret)-1:])
	}
	return "****"
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
