package logger

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/samber/lo"
)

var CommonRedactedHeaders = []string{
	"Authorization*", "Bearer*", "Session*", "*SessionId*", "*Sessid*", "*Cookie", "*Token", "*-Secret", "*-Key",
	"Password", "Passwd", "Pwd",
}

// SensitiveKeys are substrings that mark a body/query key as a secret. "sessionid"
// and "sessid" cover the common session identifiers (JSESSIONID, PHPSESSID,
// ASP.NET_SessionId, x-sessionid) without redacting unrelated "session*" fields.
var SensitiveKeys = []string{"user", "pass", "secret", "key", "token", "username", "password", "authorization", "sessionid", "sessid"}

var NonSensitiveKeys = []string{"token_type", "grant_type"}

var inlineSecretPattern = regexp.MustCompile(`(?i)(["']?)([a-z0-9_.-]*(?:user|pass|secret|key|token|sessionid|sessid|authorization)[a-z0-9_.-]*)(["']?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;&}]+)`)
var inlineAuthorizationPattern = regexp.MustCompile(`(?i)\b(authorization)(\s*[:=]\s*)([A-Za-z]+)\s+([^\s,;]+)`)

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

func IsSensitiveLogKey(v string) bool {
	v = strings.Trim(strings.TrimSpace(strings.ToLower(v)), " _-.'\"{}[]")
	for _, k := range NonSensitiveKeys {
		if v == k {
			return false
		}
	}
	switch v {
	case "authorization", "apikey", "api_key", "key", "pass", "passwd", "password", "pwd", "secret", "sessid", "sessionid", "token", "username":
		return true
	}
	for _, part := range strings.FieldsFunc(v, func(r rune) bool { return r == '_' || r == '-' || r == '.' }) {
		switch part {
		case "authorization", "key", "pass", "passwd", "password", "pwd", "secret", "sessid", "sessionid", "token":
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
		return fmt.Sprintf("****,length=%d", len(s))
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

// RedactLogMessage removes common inline secret patterns from log messages.
func RedactLogMessage(text string) string {
	return StripSecrets(text)
}

// StripSecrets takes a URL, YAML, INI, or log-formatted text and removes any potentially secret data.
// Short redacted values keep a tiny prefix/suffix to aid troubleshooting; long values only keep length.
func StripSecrets(text string) string {
	if text == "" {
		return ""
	}
	if uri, ok := parseAbsoluteURL(text); ok {
		redactURLQuery(uri)
		return uri.Redacted()
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if redacted, ok := redactSensitiveLine(line); ok {
			lines[i] = redacted
			continue
		}
		lines[i] = redactInlineSecrets(line)
	}
	return strings.Join(lines, "\n")
}

func parseAbsoluteURL(text string) (*url.URL, bool) {
	uri, err := url.Parse(text)
	return uri, err == nil && uri.Scheme != "" && uri.Host != ""
}

func redactURLQuery(uri *url.URL) {
	values := uri.Query()
	changed := false
	for key, vals := range values {
		if !IsSensitiveLogKey(key) {
			continue
		}
		for i, v := range vals {
			vals[i] = PrintableSecret(v)
		}
		values[key] = vals
		changed = true
	}
	if changed {
		uri.RawQuery = values.Encode()
	}
}

func redactSensitiveLine(line string) (string, bool) {
	idx := strings.IndexAny(line, ":=")
	if idx <= 0 {
		return "", false
	}
	key := strings.Trim(line[:idx], " \t\"'{[")
	if !IsSensitiveLogKey(key) {
		return "", false
	}

	rest := line[idx+1:]
	trimmed := strings.TrimLeft(rest, " \t")
	spaces := rest[:len(rest)-len(trimmed)]
	if trimmed == "" {
		return line, true
	}
	value, suffix := splitSecretValue(trimmed)
	return line[:idx+1] + spaces + redactSecretLiteral(value) + suffix, true
}

func splitSecretValue(s string) (value, suffix string) {
	if len(s) == 0 {
		return "", ""
	}
	if s[0] == '\'' || s[0] == '"' {
		if end := strings.IndexByte(s[1:], s[0]); end >= 0 {
			idx := end + 2
			return s[:idx], s[idx:]
		}
	}
	if idx := strings.IndexAny(s, ",;&"); idx >= 0 {
		return strings.TrimRight(s[:idx], " \t"), s[idx:]
	}
	return s, ""
}

func redactSecretLiteral(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return string(s[0]) + PrintableSecret(s[1:len(s)-1]) + string(s[len(s)-1])
	}
	return PrintableSecret(s)
}

func redactInlineSecrets(line string) string {
	line = inlineAuthorizationPattern.ReplaceAllStringFunc(line, func(match string) string {
		parts := inlineAuthorizationPattern.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		return parts[1] + parts[2] + parts[3] + " " + printableValue(parts[4])
	})
	return inlineSecretPattern.ReplaceAllStringFunc(line, func(match string) string {
		parts := inlineSecretPattern.FindStringSubmatch(match)
		if len(parts) != 5 || !IsSensitiveLogKey(parts[2]) {
			return match
		}
		return parts[1] + parts[2] + parts[3] + redactSecretLiteral(parts[4])
	})
}
