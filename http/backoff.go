package http

import (
	"math"
	"time"
)

func exponentialBackoff(config *RetryConfig, retriesRemaining uint) time.Duration {
	factor := math.Pow(config.Factor, float64(config.Total-retriesRemaining))
	// grow backoff time exponentially as the retryCount approaches zero
	sleepDuration := config.RetryWait * time.Duration(factor)

	time.Sleep(sleepDuration)
	return sleepDuration
}
