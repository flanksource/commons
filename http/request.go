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
)

// Request represents an HTTP request that can be customized and executed.
// It provides a fluent API for setting headers, query parameters, body, and other options.
// Request instances should be created using Client.R(ctx).
type Request struct {
	ctx         context.Context
	client      *Client
	retryConfig RetryConfig
	method      string
	rawURL      string
	url         *url.URL
	body        io.Reader
	headers     http.Header
	queryParams url.Values
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
		r.body = t
	case []byte:
		buf := bytes.Buffer{}
		buf.Write(t)
		r.body = &buf
	case string:
		r.body = strings.NewReader(t)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return r.Body(b)
	}

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
	var retriesRemaining = r.retryConfig.MaxRetries
	for {
		response, err := r.client.roundTrip(r)
		if err != nil {
			if retriesRemaining <= 0 {
				return nil, err
			}

			retriesRemaining--
			exponentialBackoff(r.retryConfig, retriesRemaining)
			continue
		}

		return response, nil
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
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %s\n", r.method, logger.StripSecrets(r.url.String())))
	for k, v := range logger.StripSecretsFromMap(r.HeaderMap()) {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", console.Grayf(k), v))
	}
	body, _ := io.ReadAll(r.body)
	sb.WriteString(logger.StripSecrets(string(body)))
	return sb.String()
}
