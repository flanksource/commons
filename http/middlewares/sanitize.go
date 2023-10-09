package middlewares

import (
	"net/http"

	"github.com/flanksource/commons/collections"
)

var commonRedactedHeaders = []string{
	"Authorization*", "Bearer*", "Session*", "*Cookie", "*Token", "*-Secret", "*-Key",
	"Password", "Passwd", "Pwd",
}

const redactedString = "████████████████████"

func SanitizeHeaders(headers http.Header, redactedHeaders ...string) http.Header {
	var redacted = http.Header{}

	redactedHeaders = append(redactedHeaders, commonRedactedHeaders...)

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
