package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/commons/text"
)

const defaultMaxBufferSize = 4 * 1024 * 1024 // 4 MB

// MaxBufferSizeProperty caps how many bytes of an io.Reader request body are
// buffered for retry replay. Bodies larger than this stream through once and
// cannot be retried. Set -P http.request.maxBufferSize=0 to disable the cap.
const MaxBufferSizeProperty = "http.request.maxBufferSize"

// Request represents an HTTP request that can be customized and executed.
// It provides a fluent API for setting headers, query parameters, body, and other options.
// Request instances should be created using Client.R(ctx).
type Request struct {
	ctx           context.Context
	client        *Client
	retryConfig   RetryConfig
	retryStrategy RetryStrategy
	method        string
	rawURL        string
	url           *url.URL
	body          io.Reader
	bodyBytes     []byte
	bodyBuffered  bool
	headers       http.Header
	queryParams   url.Values
}

func (r *Request) GetHeaders() map[string]string {
	return toMap(r.headers)
}

func (r *Request) GetHeader(key string) string {
	if r.headers == nil {
		return ""
	}

	return r.headers.Get(key)
}

// Header sets a header for the request.
// Multiple calls with the same key will overwrite previous values.
//
// Example:
//
//	resp, err := client.R(ctx).
//		Header("Authorization", "Bearer token").
//		Header("Content-Type", "application/json").
//		GET("/api/data")
func (r *Request) Header(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// QueryParam sets a query parameter for the request.
// Multiple calls with the same key will overwrite previous values.
//
// Example:
//
//	resp, err := client.R(ctx).
//		QueryParam("page", "1").
//		QueryParam("limit", "10").
//		GET("/api/users")
func (r *Request) QueryParam(key, value string) *Request {
	r.queryParams.Set(key, value)
	return r
}

// QueryParamAdd adds a value to a query parameter.
// Unlike QueryParam, this allows multiple values for the same key.
//
// Example:
//
//	// Results in: /api/search?tag=go&tag=http
//	resp, err := client.R(ctx).
//		QueryParamAdd("tag", "go").
//		QueryParamAdd("tag", "http").
//		GET("/api/search")
func (r *Request) QueryParamAdd(key, value string) *Request {
	r.queryParams.Add(key, value)
	return r
}

// Get performs an HTTP GET request to the specified URL.
//
// Example:
//
//	resp, err := client.R(ctx).Get("https://api.example.com/users")
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
func (r *Request) Get(url string) (*Response, error) {
	return r.Do(http.MethodGet, url)
}

// Post performs an HTTP POST request with the given body.
// The body can be a string, []byte, io.Reader, or any type that can be JSON marshaled.
//
// Example:
//
//	user := map[string]string{"name": "John", "email": "john@example.com"}
//	resp, err := client.R(ctx).
//		Header("Content-Type", "application/json").
//		Post("https://api.example.com/users", user)
func (r *Request) Post(url string, body any) (*Response, error) {
	if err := r.Body(body); err != nil {
		return nil, fmt.Errorf("error setting body: %w", err)
	}
	return r.Do(http.MethodPost, url)
}

func (r *Request) Put(url string, body any) (*Response, error) {
	if err := r.Body(body); err != nil {
		return nil, fmt.Errorf("error setting body: %w", err)
	}
	return r.Do(http.MethodPut, url)
}

func (r *Request) Patch(url string, body any) (*Response, error) {
	if err := r.Body(body); err != nil {
		return nil, fmt.Errorf("error setting body: %w", err)
	}
	return r.Do(http.MethodPatch, url)
}

func (r *Request) Delete(url string) (*Response, error) {
	return r.Do(http.MethodDelete, url)
}

// Retry configures retry behavior for this specific request,
// overriding the client's default retry configuration.
//
// Parameters:
//   - maxRetries: Maximum number of retry attempts
//   - baseDuration: Initial delay between retries
//   - exponent: Multiplier for exponential backoff
//
// Example:
//
//	// Retry this request up to 5 times
//	resp, err := client.R(ctx).
//		Retry(5, 500*time.Millisecond, 2.0).
//		GET("/flaky-endpoint")
func (r *Request) Retry(maxRetries uint, baseDuration time.Duration, exponent float64) *Request {
	r.retryConfig.MaxRetries = maxRetries
	r.retryConfig.RetryWait = baseDuration
	r.retryConfig.Factor = exponent
	return r
}

// RetryStrategy installs a per-request retry callback, overriding any
// strategy configured on the client. See Client.RetryStrategy for the full
// contract; pass nil to clear an inherited strategy and fall back to the
// legacy Retry()/RetryConfig path for this request.
func (r *Request) RetryStrategy(fn RetryStrategy) *Request {
	r.retryStrategy = fn
	return r
}

// Body sets the request body. Accepts multiple types:
//   - io.Reader: Used directly as the body
//   - []byte: Wrapped in a bytes.Reader
//   - string: Converted to []byte and wrapped
//   - Any other type: JSON marshaled
//
// Example:
//
//	// JSON body
//	req.Body(map[string]string{"key": "value"})
//
//	// String body
//	req.Body("raw text data")
//
//	// Reader body
//	req.Body(bytes.NewReader(data))
func (r *Request) Body(v any) error {
	switch t := v.(type) {
	case io.Reader:
		// A raw reader is single-use; buffer it once (up to maxBufferSize) so a
		// retried attempt can replay the same bytes instead of sending an empty
		// body. Bodies over the cap stream through un-buffered and cannot retry.
		limit := properties.Int(defaultMaxBufferSize, MaxBufferSizeProperty)
		if limit <= 0 {
			b, err := io.ReadAll(t)
			if err != nil {
				return err
			}
			r.setBodyBytes(b)
			return nil
		}
		// Read one byte past the cap to detect an over-limit reader without
		// pulling the whole thing into memory.
		prefix, err := io.ReadAll(io.LimitReader(t, int64(limit)+1))
		if err != nil {
			return err
		}
		if len(prefix) <= limit {
			r.setBodyBytes(prefix)
			return nil
		}
		logger.Debugf("request body exceeds %s buffer cap (%s); streaming un-buffered, retries disabled for this request",
			text.HumanizeBytes(limit), MaxBufferSizeProperty)
		r.body = io.MultiReader(bytes.NewReader(prefix), t)
		r.bodyBuffered = false
	case []byte:
		r.setBodyBytes(t)
	case string:
		r.setBodyBytes([]byte(t))
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return r.Body(b)
	}

	return nil
}

// setBodyBytes records the request body as a replayable byte slice and points
// r.body at a fresh reader over it. Retries call resetBody to rewind.
func (r *Request) setBodyBytes(b []byte) {
	r.bodyBytes = b
	r.bodyBuffered = true
	r.body = bytes.NewReader(b)
}

// resetBody rewinds the request body before a retried attempt. roundTrip drains
// r.body, so without this every retry of a body-carrying request would send an
// empty body (a downstream JSON validation failure). A no-op when the body
// was never buffered (e.g. a bodiless GET).
func (r *Request) resetBody() {
	if r.bodyBuffered {
		r.body = bytes.NewReader(r.bodyBytes)
	}
}

// prepareRetry rewinds a buffered body for replay, or refuses the retry when
// the body was streamed un-buffered (over the maxBufferSize cap) — resending a
// drained stream would silently transmit an empty body.
func (r *Request) prepareRetry() error {
	if !r.bodyBuffered && r.body != nil {
		return fmt.Errorf("cannot retry request: body exceeded %s and was streamed un-buffered (raise the cap to enable retries)", MaxBufferSizeProperty)
	}
	r.resetBody()
	return nil
}

// Do performs an HTTP request with the specified method and URL.
func (r *Request) Do(method, reqURL string) (resp *Response, err error) {
	r.method = method
	r.rawURL = reqURL

	r.url, err = url.Parse(r.rawURL)
	if err != nil {
		return nil, err
	}

	if !r.url.IsAbs() {
		tempURL := r.url.String()
		if len(tempURL) > 0 && tempURL[0] != '/' {
			tempURL = "/" + tempURL
		}

		r.url, err = url.Parse(r.client.baseURL + tempURL)
		if err != nil {
			return nil, err
		}
	}

	return r.do()
}

func (r *Request) do() (resp *Response, err error) {
	if r.retryStrategy != nil {
		return r.doWithStrategy()
	}

	var retriesRemaining = r.retryConfig.MaxRetries
	for {
		response, err := r.client.roundTrip(r)
		if response == nil {
			response = &Response{}
		}
		if response.Request == nil {
			response.Request = r
		}
		if err != nil {
			if retriesRemaining <= 0 {
				return nil, err
			}

			retriesRemaining--
			exponentialBackoff(r.retryConfig, retriesRemaining)
			if err := r.prepareRetry(); err != nil {
				return nil, err
			}
			continue
		}

		return response, nil
	}
}

// doWithStrategy runs the request loop under a caller-supplied RetryStrategy.
// The strategy is asked after every attempt — including the final one — and
// owns the attempt cap. The legacy RetryConfig path is bypassed entirely.
func (r *Request) doWithStrategy() (*Response, error) {
	for attempt := 0; ; attempt++ {
		response, err := r.client.roundTrip(r)
		if response == nil {
			response = &Response{}
		}
		if response.Request == nil {
			response.Request = r
		}

		retry, delay := r.retryStrategy(response, err, attempt)
		if !retry {
			if err != nil {
				return nil, err
			}
			return response, nil
		}

		if delay > 0 {
			select {
			case <-r.ctx.Done():
				return nil, r.ctx.Err()
			case <-time.After(delay):
			}
		}
		if err := r.prepareRetry(); err != nil {
			return nil, err
		}
	}
}

func (r *Request) HeaderMap() map[string]string {
	headers := make(map[string]string)
	for k, v := range r.headers {
		headers[k] = strings.Join(v, ", ")
	}
	return headers
}

func (r *Request) Debug() string {
	if r == nil {
		return "<nil request>"
	}
	var sb strings.Builder
	sb.WriteString(r.method)
	if r.url != nil {
		sb.WriteString(" " + r.url.String() + "\n")
	} else if r.client != nil && r.client.baseURL != "" {
		sb.WriteString(r.client.baseURL + "\n")
	} else {
		sb.WriteString(" <nil url>\n")
	}
	for k, v := range logger.StripSecretsFromMap(r.HeaderMap()) {
		fmt.Fprintf(&sb, "  %s: %s\n", console.Grayf("%s", k), v)
	}
	if !r.client.authConfig.IsEmpty() {
		sb.WriteString("  " + console.Grayf("%s", "Authorization: ") + r.client.authConfig.Username + ":" + logger.PrintableSecret(r.client.authConfig.Password) + "\n")
	}
	sb.WriteString(logger.StripSecrets(string(r.bodyBytes)))
	return sb.String()
}
