package http

import (
	"context"
	"net"
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
