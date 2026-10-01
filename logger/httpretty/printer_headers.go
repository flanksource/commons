package httpretty

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/flanksource/commons/logger/httpretty/internal/color"
	"github.com/flanksource/commons/logger/httpretty/internal/header"
)

func (p *printer) printHeaders(prefix rune, h http.Header) {
	if !p.logger.SkipSanitize {
		h = header.Sanitize(header.DefaultSanitizers, h)
		h = sanitizeRedactedHeaders(h, p.logger.RedactedHeaders)
	}

	longest, sorted := sortHeaderKeys(h, p.logger.cloneSkipHeader())
	for _, key := range sorted {
		for _, v := range h[key] {
			var pad string
			if p.logger.Align {
				pad = strings.Repeat(" ", longest-len(key))
			}
			p.printf("%c %s%s %s%s\n", prefix,
				p.format(color.FgBlue, color.Bold, key),
				p.format(color.FgRed, ":"),
				pad,
				p.format(color.FgYellow, v))
		}
	}
}

func sanitizeRedactedHeaders(h http.Header, patterns []string) http.Header {
	if len(patterns) == 0 {
		return h
	}

	out := h.Clone()
	for key, values := range out {
		if !matchHeaderPattern(key, patterns...) {
			continue
		}
		redacted := make([]string, 0, len(values))
		for _, value := range values {
			redacted = append(redacted, redactHeaderValue(value))
		}
		out[key] = redacted
	}
	return out
}

func matchHeaderPattern(item string, patterns ...string) bool {
	itemLower := strings.ToLower(http.CanonicalHeaderKey(item))
	for _, pattern := range patterns {
		patternLower := strings.ToLower(http.CanonicalHeaderKey(strings.TrimSpace(pattern)))
		switch {
		case patternLower == "" || patternLower == "!":
			continue
		case patternLower == "*":
			return true
		case strings.HasPrefix(patternLower, "*") && strings.HasSuffix(patternLower, "*"):
			if strings.Contains(itemLower, strings.Trim(patternLower, "*")) {
				return true
			}
		case strings.HasPrefix(patternLower, "*"):
			if strings.HasSuffix(itemLower, strings.TrimPrefix(patternLower, "*")) {
				return true
			}
		case strings.HasSuffix(patternLower, "*"):
			if strings.HasPrefix(itemLower, strings.TrimSuffix(patternLower, "*")) {
				return true
			}
		case itemLower == patternLower:
			return true
		}
	}
	return false
}

func redactHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	if scheme, credential, ok := strings.Cut(value, " "); ok && credential != "" {
		return scheme + " " + strings.Repeat("█", 8)
	}
	return strings.Repeat("█", 8)
}

func sortHeaderKeys(h http.Header, skipped map[string]struct{}) (int, []string) {
	var (
		keys    = make([]string, 0, len(h))
		longest int
	)
	for key := range h {
		if _, skip := skipped[key]; skip {
			continue
		}
		keys = append(keys, key)
		if l := len(key); l > longest {
			longest = l
		}
	}
	sort.Strings(keys)
	if i := slices.Index(keys, "Host"); i > -1 {
		keys = append([]string{"Host"}, slices.Delete(keys, i, i+1)...)
	}
	return longest, keys
}

func (p *printer) printRequestHeader(req *http.Request) {
	uri := req.URL.String()
	if req.URL.Host == "" {
		host := req.Host
		if host == "" {
			host = req.Header.Get("Host")
		}
		scheme := req.URL.Scheme
		if scheme == "" {
			if req.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		uri = fmt.Sprintf("%s://%s%s", scheme, host, req.URL.RequestURI())
	}

	proto := ""
	if req.Proto != "" && req.Proto != "HTTP/1.1" {
		proto = " " + p.format(color.FgBlue, req.Proto)
	}

	p.printf("%s %s%s\n",
		p.format(color.FgBlue, color.Bold, req.Method),
		p.format(color.FgYellow, uri),
		proto)
	p.printHeaders('>', addRequestHeaders(req))
	p.println()
}

func addRequestHeaders(req *http.Request) http.Header {
	cp := http.Header{}
	for k, v := range req.Header {
		cp[k] = v
	}

	if len(req.Header.Values("Content-Length")) == 0 && req.ContentLength > 0 {
		cp.Set("Content-Length", fmt.Sprintf("%d", req.ContentLength))
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	if host != "" && host != req.URL.Host {
		cp.Set("Host", host)
	}
	return cp
}
