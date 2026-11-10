package logger

import (
	"encoding/base64"
	"strings"
)

// minRedactedCredentialLength is the shortest credential replaced in free
// text: shorter values would match unrelated characters and corrupt the text.
const minRedactedCredentialLength = 4

// RedactAuthorization removes credentials echoed outside recognized secret
// fields, including decoded Basic-auth passwords in HTTP error responses.
// Credentials shorter than minRedactedCredentialLength are left in place.
func RedactAuthorization(text, authorization string) string {
	parts := strings.Fields(authorization)
	if len(parts) == 0 {
		return text
	}
	credential := parts[len(parts)-1]
	text = redactCredential(text, credential)
	if len(parts) == 2 && strings.EqualFold(parts[0], "basic") {
		decoded, err := base64.StdEncoding.DecodeString(credential)
		if err != nil {
			return text
		}
		text = redactCredential(text, string(decoded))
		if _, password, ok := strings.Cut(string(decoded), ":"); ok {
			text = redactCredential(text, password)
		}
	}
	return text
}

func redactCredential(text, credential string) string {
	if len(credential) < minRedactedCredentialLength {
		return text
	}
	return strings.ReplaceAll(text, credential, "[REDACTED]")
}
