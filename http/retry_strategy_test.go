package http

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryStrategy_CalledPerAttempt(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(stdhttp.StatusInternalServerError)
	}))
	defer srv.Close()

	var observedAttempts []int
	strategy := func(resp *Response, err error, attempt int) (bool, time.Duration) {
		observedAttempts = append(observedAttempts, attempt)
		return attempt < 2, 0
	}

	client := NewClient().RetryStrategy(strategy)
	resp, err := client.R(context.Background()).Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 server hits, got %d", got)
	}
	if want := []int{0, 1, 2}; !equalInts(observedAttempts, want) {
		t.Fatalf("expected attempt indices %v, got %v", want, observedAttempts)
	}
}

func TestRetryStrategy_StopImmediately(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(stdhttp.StatusBadGateway)
	}))
	defer srv.Close()

	strategy := func(resp *Response, err error, attempt int) (bool, time.Duration) {
		return false, 0
	}
	client := NewClient().RetryStrategy(strategy)
	resp, err := client.R(context.Background()).Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected 1 server hit, got %d", got)
	}
}

func TestRetryStrategy_HonorsDelay(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusTooManyRequests)
	}))
	defer srv.Close()

	delay := 60 * time.Millisecond
	strategy := func(resp *Response, err error, attempt int) (bool, time.Duration) {
		if attempt >= 1 {
			return false, 0
		}
		return true, delay
	}

	start := time.Now()
	client := NewClient().RetryStrategy(strategy)
	if _, err := client.R(context.Background()).Get(srv.URL); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < delay {
		t.Fatalf("expected elapsed >= %v, got %v", delay, elapsed)
	}
}

func TestRetryStrategy_ContextCancelDuringSleep(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(stdhttp.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	strategy := func(resp *Response, err error, attempt int) (bool, time.Duration) {
		cancel()
		return true, 5 * time.Second
	}

	client := NewClient().RetryStrategy(strategy)
	_, err := client.R(ctx).Get(srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected exactly 1 server hit before cancel, got %d", got)
	}
}

func TestRetryOnStatus_RetriesAndStopsOnSuccess(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(stdhttp.StatusTooManyRequests)
			return
		}
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := NewClient().RetryStrategy(RetryOnStatus(5, time.Millisecond, stdhttp.StatusTooManyRequests))
	resp, err := client.R(context.Background()).Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 server hits, got %d", got)
	}
}

func TestRetryOnStatus_StopsAtMaxAttempts(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(stdhttp.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := NewClient().RetryStrategy(RetryOnStatus(3, time.Microsecond, stdhttp.StatusServiceUnavailable))
	resp, err := client.R(context.Background()).Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 server hits, got %d", got)
	}
}

func TestRetryOnStatus_HonorsRetryAfterSeconds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(stdhttp.StatusTooManyRequests)
			return
		}
		w.WriteHeader(stdhttp.StatusOK)
	}))
	defer srv.Close()

	// baseDelay is huge so exponential backoff would dominate if Retry-After
	// were not honored. Retry-After=1 should pick 1s, not the huge baseDelay.
	client := NewClient().RetryStrategy(RetryOnStatus(3, time.Hour, stdhttp.StatusTooManyRequests))
	start := time.Now()
	resp, err := client.R(context.Background()).Get(srv.URL)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if elapsed < 900*time.Millisecond || elapsed > 3*time.Second {
		t.Fatalf("expected ~1s wait via Retry-After, got %v", elapsed)
	}
}

func TestRetryOnStatus_DoesNotRetryUnlistedStatus(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(stdhttp.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewClient().RetryStrategy(RetryOnStatus(5, time.Microsecond, stdhttp.StatusTooManyRequests))
	resp, err := client.R(context.Background()).Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected 1 server hit, got %d", got)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		in     string
		want   time.Duration
		wantOK bool
	}{
		{"", 0, false},
		{"3", 3 * time.Second, true},
		{"0", 0, true},
		{"-5", 0, false},
		{"not-a-number", 0, false},
		{"Thu, 01 Jan 2026 12:00:05 GMT", 5 * time.Second, true},
		{"Thu, 01 Jan 2026 11:59:55 GMT", 0, true}, // past => 0
	}
	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.in, now)
		if ok != tt.wantOK {
			t.Errorf("parseRetryAfter(%q) ok=%v, want %v", tt.in, ok, tt.wantOK)
			continue
		}
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
