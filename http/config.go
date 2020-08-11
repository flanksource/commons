package http

import (
	"context"
	"net"
	"time"

	"github.com/flanksource/commons/logger"
)

// Config holds all configuration for the HTTP client
type Config struct {
	// TLS settings
	InsecureSkipVerify bool

	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	ProxyHost string
	ProxyPort uint16

	// Headers are automatically added to all requests
	Headers map[string]string

	// BaseURL is added as a prefix to all URLs
	BaseURL string

	Logger logger.Logger

	// RETRY HANDLING
	// Cancel the request after this timeout
	Timeout time.Duration
	// Number of retries to attempt
	Retries uint
	// Time to wait between retries
	RetryWait time.Duration
	// Amount to increase RetryWait with each failure, 2.0 is a good option for exponential backoff
	Factor float64
}
