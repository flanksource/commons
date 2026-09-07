package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
)

// DefensiveOptions controls which network destinations a defensive client may reach.
type DefensiveOptions struct {
	Public    bool
	Private   bool
	Localhost bool
	Proxy     bool
	Redirect  bool
}

// NewDefensive creates a client that can reach public HTTP(S) destinations only.
func NewDefensive() *Client {
	return NewClient().
		Use(Defensive(DefensiveOptions{Public: true})).
		RedirectPolicy(0)
}

// Defensive returns transport middleware that restricts requests to the
// destination classes explicitly enabled by options. It must wrap an
// *http.Transport before middleware that hides the concrete transport.
func Defensive(options DefensiveOptions) middlewares.Middleware {
	return func(next stdhttp.RoundTripper) stdhttp.RoundTripper {
		if next == nil {
			next = stdhttp.DefaultTransport
		}
		transport, ok := next.(*stdhttp.Transport)
		if !ok {
			// Defensive controls the dialer, so it can only wrap the concrete
			// transport. Report the misordering from RoundTrip: a builder
			// method has no way to return an error, and failing the request is
			// safer than panicking during client setup.
			return misconfiguredRoundTripper{err: fmt.Errorf(
				"defensive HTTP middleware must wrap an *http.Transport, got %T: register it before middleware that replaces the transport", next)}
		}

		configured := transport.Clone()
		dialer := configured.DialContext
		if dialer == nil {
			dialer = (&net.Dialer{}).DialContext
		}
		configured.DialContext = defensiveDialContext(options, net.DefaultResolver, networkDialerFunc(dialer))
		if options.Proxy {
			if configured.Proxy == nil {
				configured.Proxy = stdhttp.ProxyFromEnvironment
			}
		} else {
			configured.Proxy = nil
		}
		if configured.TLSClientConfig == nil {
			configured.TLSClientConfig = &tls.Config{}
		} else {
			configured.TLSClientConfig = configured.TLSClientConfig.Clone()
		}
		configured.TLSClientConfig.InsecureSkipVerify = false
		configured.DialTLSContext = defensiveTLSDialContext(configured.DialContext, configured.TLSClientConfig)

		return &defensiveRoundTripper{transport: configured, options: options, resolver: net.DefaultResolver}
	}
}

// misconfiguredRoundTripper turns a configuration error into a request error,
// so a client built with the wrong middleware order fails every request with an
// explicit message instead of panicking.
type misconfiguredRoundTripper struct {
	err error
}

func (t misconfiguredRoundTripper) RoundTrip(*stdhttp.Request) (*stdhttp.Response, error) {
	return nil, t.err
}

type defensiveRoundTripper struct {
	transport *stdhttp.Transport
	options   DefensiveOptions
	resolver  defensiveResolver
}

func (t *defensiveRoundTripper) RoundTrip(request *stdhttp.Request) (*stdhttp.Response, error) {
	if err := validateDefensiveURL(request.URL, t.options); err != nil {
		return nil, err
	}
	if request.Response != nil && !t.options.Redirect {
		return nil, fmt.Errorf("defensive HTTP middleware refuses redirect to %q", request.URL)
	}
	if err := validateDefensiveCredentials(request); err != nil {
		return nil, err
	}
	if err := t.validateProxiedDestination(request); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(request)
}

// validateProxiedDestination resolves and checks the requested host when the
// connection goes through a proxy. A proxied dial only ever sees the proxy's
// address, so without this the destination policy is unenforced and the proxy
// can be asked to reach a private or metadata address.
func (t *defensiveRoundTripper) validateProxiedDestination(request *stdhttp.Request) error {
	if !t.options.Proxy || t.transport.Proxy == nil {
		return nil
	}
	proxyURL, err := t.transport.Proxy(request)
	if err != nil {
		return fmt.Errorf("resolve proxy for %q: %w", request.URL, err)
	}
	if proxyURL == nil {
		// Direct connection: defensiveDialContext validates the destination.
		return nil
	}

	host := request.URL.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		return validateDefensiveIP(ip, t.options)
	}
	if isMetadataHost(host) {
		return fmt.Errorf("defensive HTTP client refuses metadata host %q", host)
	}
	addresses, err := t.resolver.LookupIPAddr(request.Context(), host)
	if err != nil {
		return fmt.Errorf("resolve defensive HTTP host %q: %w", host, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("resolve defensive HTTP host %q: no addresses returned", host)
	}
	for _, resolved := range addresses {
		if err := validateDefensiveIP(resolved.IP, t.options); err != nil {
			return fmt.Errorf("resolve defensive HTTP host %q to %s: %w", host, resolved.IP, err)
		}
	}
	return nil
}

// validateDefensiveCredentials refuses to put credentials on the wire in
// cleartext. Client.roundTrip attaches Basic auth before the transport runs, so
// an http:// destination would expose the Authorization header to anything on
// the path. Loopback is exempt: that traffic never reaches a network.
func validateDefensiveCredentials(request *stdhttp.Request) error {
	if request.URL == nil || request.URL.Scheme == "https" {
		return nil
	}
	if isLoopbackHost(request.URL.Hostname()) {
		return nil
	}
	for _, header := range []string{"Authorization", "Proxy-Authorization"} {
		if request.Header.Get(header) != "" {
			return fmt.Errorf("defensive HTTP client refuses to send %s over %q to %q", header, request.URL.Scheme, request.URL.Host)
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defensiveTLSDialContext(dialContext func(context.Context, string, string) (net.Conn, error), config *tls.Config) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		connection, err := dialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		host, _, err := net.SplitHostPort(address)
		if err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("parse defensive TLS address %q: %w", address, err)
		}
		tlsConfig := config.Clone()
		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = host
		}
		tlsConnection := tls.Client(connection, tlsConfig)
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("defensive TLS handshake with %q: %w", address, err)
		}
		return tlsConnection, nil
	}
}

type defensiveResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type defensiveDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type networkDialerFunc func(context.Context, string, string) (net.Conn, error)

func (f networkDialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func defensiveDialContext(options DefensiveOptions, resolver defensiveResolver, dialer defensiveDialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse defensive dial address %q: %w", address, err)
		}

		if ip := net.ParseIP(host); ip != nil {
			if err := validateDefensiveIP(ip, options); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}

		if isMetadataHost(host) {
			return nil, fmt.Errorf("defensive HTTP client refuses metadata host %q", host)
		}
		addresses, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve defensive HTTP host %q: %w", host, err)
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("resolve defensive HTTP host %q: no addresses returned", host)
		}
		for _, resolved := range addresses {
			if err := validateDefensiveIP(resolved.IP, options); err != nil {
				return nil, fmt.Errorf("resolve defensive HTTP host %q to %s: %w", host, resolved.IP, err)
			}
		}

		var dialErrors []error
		for _, resolved := range addresses {
			resolvedAddress := net.JoinHostPort(resolved.IP.String(), port)
			conn, err := dialer.DialContext(ctx, network, resolvedAddress)
			if err == nil {
				return conn, nil
			}
			dialErrors = append(dialErrors, fmt.Errorf("dial %s: %w", resolvedAddress, err))
		}
		return nil, errors.Join(dialErrors...)
	}
}

func validateDefensiveURL(target *url.URL, options DefensiveOptions) error {
	if target == nil {
		return fmt.Errorf("defensive HTTP client requires a URL")
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("defensive HTTP client refuses URL scheme %q", target.Scheme)
	}
	if target.User != nil {
		return fmt.Errorf("defensive HTTP client refuses URL userinfo")
	}
	host := target.Hostname()
	if host == "" {
		return fmt.Errorf("defensive HTTP client requires a URL host")
	}
	if isMetadataHost(host) {
		return fmt.Errorf("defensive HTTP client refuses metadata host %q", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		return validateDefensiveIP(ip, options)
	}
	return nil
}

func validateDefensiveIP(ip net.IP, options DefensiveOptions) error {
	if ip == nil {
		return fmt.Errorf("defensive HTTP client refuses an invalid IP address")
	}
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || isMetadataIP(ip) {
		return fmt.Errorf("defensive HTTP client refuses reserved address %s", ip)
	}
	if ip.IsLoopback() {
		if options.Localhost {
			return nil
		}
		return fmt.Errorf("defensive HTTP client refuses localhost address %s", ip)
	}
	if ip.IsPrivate() || isCarrierGradeNAT(ip) {
		if options.Private {
			return nil
		}
		return fmt.Errorf("defensive HTTP client refuses private address %s", ip)
	}
	if !ip.IsGlobalUnicast() || !options.Public {
		return fmt.Errorf("defensive HTTP client refuses public address %s", ip)
	}
	return nil
}

func isCarrierGradeNAT(ip net.IP) bool {
	ipv4 := ip.To4()
	return ipv4 != nil && ipv4[0] == 100 && ipv4[1]&0xc0 == 0x40
}

func isMetadataIP(ip net.IP) bool {
	for _, raw := range []string{"100.100.100.200", "192.0.0.192", "fd00:ec2::254"} {
		if ip.Equal(net.ParseIP(raw)) {
			return true
		}
	}
	return false
}

func isMetadataHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == "metadata" || host == "metadata.google.internal" || host == "metadata.azure.internal" || host == "instance-data"
}
