package http

import (
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"
)

// RetryStrategy decides whether a completed HTTP attempt should be retried.
// It is called after every attempt with the response (may be nil on a
// transport error), the transport-level error (may be nil on a non-2xx
// HTTP response), and the zero-based attempt index.
//
// Return (true, delay) to retry after sleeping delay. A non-positive delay
// retries immediately. Return (false, _) to stop and surface the underlying
// attempt result.
//
// When set on a Client or Request via RetryStrategy(...), this callback
// fully supersedes the legacy RetryConfig-driven exponential-backoff loop:
// the strategy is responsible for its own attempt cap.
type RetryStrategy func(resp *Response, err error, attempt int) (retry bool, delay time.Duration)

// RetryOnStatus returns a RetryStrategy that retries on any of the given HTTP
// status codes (plus transport errors) up to maxAttempts total attempts,
// using exponential backoff starting at baseDelay (factor 2).
//
// On a 429 response, a Retry-After header is honored over the computed
// delay. Both delta-seconds and HTTP-date forms are supported.
func RetryOnStatus(maxAttempts int, baseDelay time.Duration, statuses ...int) RetryStrategy {
	statusSet := make(map[int]struct{}, len(statuses))
	for _, s := range statuses {
		statusSet[s] = struct{}{}
	}
	return func(resp *Response, err error, attempt int) (bool, time.Duration) {
		if attempt+1 >= maxAttempts {
			return false, 0
		}
		if err != nil {
			return true, backoff(baseDelay, attempt)
		}
		if resp == nil || resp.Response == nil {
			return false, 0
		}
		if _, ok := statusSet[resp.StatusCode]; !ok {
			return false, 0
		}
		if resp.StatusCode == stdhttp.StatusTooManyRequests {
			if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
				return true, d
			}
		}
		return true, backoff(baseDelay, attempt)
	}
}

func backoff(base time.Duration, attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	// 2^attempt; cap to avoid overflow at large attempt counts.
	shift := attempt
	if shift > 20 {
		shift = 20
	}
	return base * time.Duration(1<<uint(shift))
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if t, err := stdhttp.ParseTime(value); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}
