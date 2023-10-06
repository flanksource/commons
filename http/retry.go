package http

import "time"

type RetryConfig struct {
	// Number of retries to attempt
	MaxRetries uint

	// RetryWait specifies the base wait duration between retries
	RetryWait time.Duration

	// Amount to increase RetryWait with each failure, 2.0 is a good option for exponential backoff
	Factor float64
}
