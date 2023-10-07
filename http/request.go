package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
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
}

func (r *Request) getHeader(key string) string {
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

func (r *Request) Get(url string) (*Response, error) {
	return r.Send(http.MethodGet, url)
}

func (r *Request) Post(url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(http.MethodPost, url)
}

func (r *Request) Put(url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(http.MethodPut, url)
}

func (r *Request) Patch(url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(http.MethodPatch, url)
}

func (r *Request) Delete(url string) (*Response, error) {
	return r.Send(http.MethodDelete, url)
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

// TODO: Make this accept more types ([]byte, string, ...)
func (r *Request) setBody(v any) *Request {
	switch t := v.(type) {
	case io.Reader:
		r.body = t
	case []byte:
		buf := bytes.Buffer{}
		buf.Write(t)
		r.body = &buf
	}

	return r
}

func (r *Request) Send(method, reqURL string) (resp *Response, err error) {
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

	resp, err = r.Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *Request) Do() (resp *Response, err error) {
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
