package awssigv4

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
)

func canonicalPath(uri *url.URL, options SignOptions) string {
	value := uri.EscapedPath()
	if options.DisableURIPathEscaping {
		value = uriEncode(uri.Path, true)
	}
	if uri.Opaque != "" {
		opaque, _, _ := strings.Cut(strings.TrimPrefix(uri.Opaque, "//"), "?")
		if index := strings.IndexByte(opaque, '/'); index >= 0 {
			value = opaque[index:]
		}
	}
	if value == "" {
		value = "/"
	}
	if options.Service != "s3" && !options.DisablePathNormalization {
		trailingSlash := strings.HasSuffix(value, "/")
		value = path.Clean(value)
		if trailingSlash && value != "/" {
			value += "/"
		}
	}
	if options.Service != "s3" && !options.DisableURIPathEscaping {
		value = uriEncode(value, true)
	}
	return value
}

func canonicalQuery(raw string) (string, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return "", fmt.Errorf("parse SigV4 query: %w", err)
	}
	type pair struct{ key, value string }
	var pairs []pair
	for key, values := range query {
		for _, value := range values {
			pairs = append(pairs, pair{uriEncode(key, false), uriEncode(value, false)})
		}
	}
	slices.SortFunc(pairs, func(a, b pair) int {
		if order := strings.Compare(a.key, b.key); order != 0 {
			return order
		}
		return strings.Compare(a.value, b.value)
	})
	encoded := make([]string, len(pairs))
	for i, pair := range pairs {
		encoded[i] = pair.key + "=" + pair.value
	}
	return strings.Join(encoded, "&"), nil
}

func uriEncode(value string, preserveSlash bool) string {
	const hex = "0123456789ABCDEF"
	var encoded strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~", rune(c)) || c == '/' && preserveSlash {
			encoded.WriteByte(c)
		} else {
			encoded.WriteByte('%')
			encoded.WriteByte(hex[c>>4])
			encoded.WriteByte(hex[c&15])
		}
	}
	return encoded.String()
}

func canonicalHeaders(request *http.Request) (string, string) {
	host := request.Host
	if host == "" {
		host = request.URL.Host
	}
	if hostname, port, err := net.SplitHostPort(host); err == nil && (request.URL.Scheme == "https" && port == "443" || request.URL.Scheme == "http" && port == "80") {
		host = hostname
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
		request.Host = host
	}
	headers := map[string][]string{"host": {host}}
	if request.ContentLength > 0 {
		headers["content-length"] = []string{strconv.FormatInt(request.ContentLength, 10)}
	}
	headerKeys := make([]string, 0, len(request.Header))
	for key := range request.Header {
		headerKeys = append(headerKeys, key)
	}
	slices.Sort(headerKeys)
	for _, key := range headerKeys {
		values := request.Header[key]
		key = strings.ToLower(key)
		switch key {
		case "authorization", "user-agent", "x-amzn-trace-id", "expect", "transfer-encoding", "content-length", "host":
			continue
		}
		headers[key] = append(headers[key], values...)
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var canonical strings.Builder
	for _, key := range keys {
		values := make([]string, len(headers[key]))
		for i, value := range headers[key] {
			values[i] = strings.Join(strings.Fields(value), " ")
		}
		canonical.WriteString(key + ":" + strings.Join(values, ",") + "\n")
	}
	return canonical.String(), strings.Join(keys, ";")
}
