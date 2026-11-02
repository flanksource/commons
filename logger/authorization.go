package logger

import (
	"encoding/base64"
	"strings"
)

// RedactAuthorization removes credentials echoed outside recognized secret
// fields, including decoded Basic-auth passwords in HTTP error responses.
func RedactAuthorization(text, authorization string) string {
	parts := strings.Fields(authorization)
	if len(parts) == 0 {
		return text
	}
	credential := parts[len(parts)-1]
	text = strings.ReplaceAll(text, credential, "[REDACTED]")
	if len(parts) == 2 && strings.EqualFold(parts[0], "basic") {
		decoded, err := base64.StdEncoding.DecodeString(credential)
		if err != nil {
			return text
		}
		text = strings.ReplaceAll(text, string(decoded), "[REDACTED]")
		if _, password, ok := strings.Cut(string(decoded), ":"); ok && password != "" {
			text = strings.ReplaceAll(text, password, "[REDACTED]")
		}
	}
	return text
}
