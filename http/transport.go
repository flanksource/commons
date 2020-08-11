package http

import (
	"crypto/tls"
	"net/http"
)

// createHTTPTransport creates an HTTP transport from the given configuration
func createHTTPTransport(config *Config) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport)

	// configure proxy
	proxy, err := configureProxy(config)
	if err != nil {
		config.Logger.Warnf("cannot configure proxy: %v", err)
	} else {
		transport.Proxy = proxy
	}

	// configure TLS certificate verification skipping
	if config.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// configure dial context
	transport.DialContext = config.DialContext

	return transport
}
