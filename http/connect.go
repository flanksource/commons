package http

import (
	"context"
	"fmt"
	"net"
	stdhttp "net/http"
	"time"
)

// ConnectTimeout limits the time spent establishing a network connection.
func (c *Client) ConnectTimeout(timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("connect timeout must be greater than zero, got %s", timeout)
	}

	base := c.httpClient.Transport
	if base == nil {
		base = stdhttp.DefaultTransport
	}
	transport, ok := base.(*stdhttp.Transport)
	if !ok {
		return nil, fmt.Errorf("configure connect timeout: transport %T is not an *http.Transport", base)
	}

	configured := transport.Clone()
	dialContext := configured.DialContext
	if dialContext == nil {
		dialContext = (&net.Dialer{}).DialContext
	}
	configured.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		connectCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return dialContext(connectCtx, network, address)
	}
	c.httpClient.Transport = configured
	return c, nil
}
