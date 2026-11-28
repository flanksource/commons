package httppolicy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// AllowsCredentials permits HTTPS and HTTP to loopback hosts.
func AllowsCredentials(target *url.URL) bool {
	if target == nil || target.Hostname() == "" {
		return false
	}
	return target.Scheme == "https" || (target.Scheme == "http" && IsLoopbackHost(target.Hostname()))
}

// IsLoopbackHost recognizes localhost and numeric loopback addresses without resolving DNS.
func IsLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// WithoutRedirects returns a copy that sends each request only to its original destination.
func WithoutRedirects(client *http.Client) http.Client {
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return noRedirect
}
