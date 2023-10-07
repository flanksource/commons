package http

import (
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

func exponentialBackoff(config *RetryConfig, retriesRemaining uint) time.Duration {
	factor := math.Pow(config.Factor, float64(config.MaxRetries-retriesRemaining))
	// grow backoff time exponentially as the retryCount approaches zero
	sleepDuration := config.RetryWait * time.Duration(factor)

	time.Sleep(sleepDuration)
	return sleepDuration
}
