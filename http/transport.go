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

	if config.Transport != nil {
		if config.Transport.TLS != nil {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}

			transport.TLSClientConfig.InsecureSkipVerify = config.Transport.TLS.InsecureSkipVerify

			transport.TLSClientConfig.ServerName = config.Transport.TLS.ServerName
		}

		transport.DisableKeepAlives = config.Transport.DisableKeepAlives

		if config.Transport.DialContext != nil {
			transport.DialContext = config.Transport.DialContext
		}
	}

	return transport
}
