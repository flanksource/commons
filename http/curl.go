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
// All headers including Authorization are included unredacted so the
// command can be copy-pasted for debugging.
func ToCurl(req *http.Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "curl -X %s '%s'", req.Method, escapeSingleQuote(req.URL.String()))

	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(&b, " -H '%s: %s'", escapeSingleQuote(k), escapeSingleQuote(strings.Join(req.Header[k], ", ")))
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
	logger.Tracef(ToCurl(req))
	return t.base.RoundTrip(req)
}
