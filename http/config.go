package http

import (
	"context"
	"net"
	"time"

	"github.com/flanksource/commons/logger"
)

type TLSConfig struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name
	InsecureSkipVerify bool

	ServerName string
}

type TransportConfig struct {
	TLS *TLSConfig

	// DisableKeepAlives prevents reuse of TCP connections
	DisableKeepAlives bool

	// DialContext specifies the dial function for creating unencrypted TCP connections.
	// If DialContext is nil (and the deprecated Dial below is also nil),
	// then the transport dials using package net.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
}

type AuthConfig struct {
	// Username for basic Auth
	Username string

	// Password for basic Auth
	Password string

	// Ntlm controls whether to use NTLM
	Ntlm bool

	// Ntlmv2 controls whether to use NTLMv2
	Ntlmv2 bool
}

type RetryConfig struct {
	// Number of retries to attempt
	Total uint

	// RetryWait specifies the base wait duration between retries
	RetryWait time.Duration

	// Amount to increase RetryWait with each failure, 2.0 is a good option for exponential backoff
	Factor float64
}

// Config holds all configuration for the HTTP client
type Config struct {
	// Auth specifies the authentication configuration
	Auth *AuthConfig

	// Transport specifies the transport configuration
	Transport *TransportConfig

	// Retries specifies the configuration for retries.
	Retries *RetryConfig

	// ConnectTo specifies the host to connect to.
	// Might be different from the host specified in the URL.
	ConnectTo string

	// Headers are automatically added to all requests
	Headers map[string]string

	// BaseURL is added as a prefix to all URLs
	BaseURL string

	// Specify if request/response bodies should be logged
	TraceBody     bool
	TraceResponse bool
	Trace         bool

	// GET's are on TRACE, PUT/PATCH/POST are on Debug, and DELETE are on Info
	Logger logger.Logger

	// Timeout specifies a time limit for requests made by this Client
	Timeout time.Duration

	// ProxyHost specifies a proxy
	ProxyHost string

	// ProxyPort specifies the proxy's port
	ProxyPort uint16

	// DNSCache specifies whether to cache DNS lookups
	DNSCache bool
}
