package http

import stdhttp "net/http"

func (c *Client) setTransport(transport *stdhttp.Transport) {
	c.tlsConfig = transport.TLSClientConfig
	c.httpClient.Transport = transport
}
