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
)

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

// Header set a header for the request.
func (r *Request) Header(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// QueryParam sets query params
func (r *Request) QueryParam(key, value string) *Request {
	r.queryParams.Set(key, value)
	return r
}

// QueryParamAdd adds value to query param key
func (r *Request) QueryParamAdd(key, value string) *Request {
	r.queryParams.Add(key, value)
	return r
}

func (r *Request) Get(url string) (*Response, error) {
	return r.Do(http.MethodGet, url)
}

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

// Retry configuration retrying on failure with exponential backoff.
//
// Base duration of a second & an exponent of 2 is a good option.
func (r *Request) Retry(maxRetries uint, baseDuration time.Duration, exponent float64) *Request {
	r.retryConfig.MaxRetries = maxRetries
	r.retryConfig.RetryWait = baseDuration
	r.retryConfig.Factor = exponent
	return r
}

// Body sets the request body
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
