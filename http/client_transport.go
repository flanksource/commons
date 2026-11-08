package http

import (
	"crypto/tls"
	stdhttp "net/http"
)

// configureTransport applies configure to a clone of the innermost
// *http.Transport behind the client's transport (http.DefaultTransport when
// unset), and re-applies any TransportWrapper layers around the clone.
func (c *Client) configureTransport(configure func(*stdhttp.Transport)) error {
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = stdhttp.DefaultTransport
	}
	transport, rewrap, err := configurableTransport(c.httpClient.Transport)
	if err != nil {
		return err
	}
	configure(transport)
	c.tlsConfig = transport.TLSClientConfig
	c.httpClient.Transport = rewrap(transport)
	return nil
}

// configureTransportOrFailRequests is configureTransport for builder methods,
// which cannot return an error: a failure replaces the transport so that every
// request fails with it.
func (c *Client) configureTransportOrFailRequests(configure func(*stdhttp.Transport)) {
	if err := c.configureTransport(configure); err != nil {
		c.httpClient.Transport = misconfiguredRoundTripper{err: err}
	}
}

func ensureTLSClientConfig(transport *stdhttp.Transport) *tls.Config {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	return transport.TLSClientConfig
}
