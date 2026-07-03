package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/flanksource/commons/logger"
)

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// ToCurl converts an http.Request into an equivalent curl command string.
// Sensitive headers are redacted before the command is logged or displayed.
func ToCurl(req *http.Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "curl -X %s '%s'", req.Method, escapeSingleQuote(req.URL.String()))

	headers := logger.SanitizeHeaders(req.Header)
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(&b, " -H '%s: %s'", escapeSingleQuote(k), escapeSingleQuote(strings.Join(headers[k], ", ")))
	}

	if req.Body != nil && req.Body != http.NoBody {
		body, err := io.ReadAll(req.Body)
		if err == nil && len(body) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(body))
			fmt.Fprintf(&b, " --data '%s'", escapeSingleQuote(string(body)))
		} else {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	return b.String()
}

type curlLogTransport struct {
	base http.RoundTripper
}

func (t *curlLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	logger.Tracef("%s", ToCurl(req))
	return t.base.RoundTrip(req)
}
