package http

import (
	"context"
	"math"
	"time"
)

type RetryConfig struct {
	// Number of retries to attempt
	MaxRetries uint

	// RetryWait specifies the base wait duration between retries
	RetryWait time.Duration

	// Amount to increase RetryWait with each failure, 2.0 is a good option for exponential backoff
	Factor float64
}

// exponentialBackoff waits out the backoff before the next retry, returning
// ctx.Err() as soon as ctx is done instead of sleeping past it.
func exponentialBackoff(ctx context.Context, config RetryConfig, retriesRemaining uint) error {
	factor := math.Pow(config.Factor, float64(config.MaxRetries-retriesRemaining))
	// grow backoff time exponentially as the retryCount approaches zero
	timer := time.NewTimer(config.RetryWait * time.Duration(factor))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
