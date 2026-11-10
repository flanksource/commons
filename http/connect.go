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

	err := c.configureTransport(func(transport *stdhttp.Transport) {
		dialContext := c.connectDialContext
		if dialContext == nil {
			dialContext = transport.DialContext
			if dialContext == nil {
				dialContext = (&net.Dialer{}).DialContext
			}
			c.connectDialContext = dialContext
		}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			connectCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return dialContext(connectCtx, network, address)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("configure connect timeout: %w", err)
	}
	return c, nil
}
