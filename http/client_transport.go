package http

import (
	"crypto/tls"
	stdhttp "net/http"
)

// configureTransport applies configure to a clone of the innermost
// *http.Transport behind the client's base transport (http.DefaultTransport
// when unset), re-applies any TransportWrapper layers around the clone, and
// rebuilds the middleware chain registered with Use on top of it.
func (c *Client) configureTransport(configure func(*stdhttp.Transport)) error {
	inner, rewrap, err := unwrapTransport(c.transportBase())
	if err != nil {
		return err
	}
	transport := inner.Clone()
	configure(transport)
	c.tlsConfig = transport.TLSClientConfig
	c.setTransportBase(rewrap(transport))
	return nil
}

// configureTransportOrFailRequests is configureTransport for builder methods,
// which cannot return an error: a failure replaces the base transport so that
// every request fails with it.
func (c *Client) configureTransportOrFailRequests(configure func(*stdhttp.Transport)) {
	if err := c.configureTransport(configure); err != nil {
		c.setTransportBase(misconfiguredRoundTripper{err: err})
	}
}

// transportBase returns the transport beneath the middleware registered with
// Use, defaulting to http.DefaultTransport.
func (c *Client) transportBase() stdhttp.RoundTripper {
	base := c.httpClient.Transport
	if len(c.transportMiddlewares) > 0 {
		base = c.baseTransport
	}
	if base == nil {
		return stdhttp.DefaultTransport
	}
	return base
}

func (c *Client) setTransportBase(base stdhttp.RoundTripper) {
	if len(c.transportMiddlewares) == 0 {
		c.httpClient.Transport = base
		return
	}
	c.baseTransport = base
	c.httpClient.Transport = applyMiddleware(base, c.transportMiddlewares...)
}

func ensureTLSClientConfig(transport *stdhttp.Transport) *tls.Config {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	return transport.TLSClientConfig
}
