package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"go.opentelemetry.io/otel/attribute"
)

type Request struct {
	ctx         context.Context
	client      *Client
	retryConfig *RetryConfig
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

func (r *Request) SetContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// SetHeader set a header for the request.
func (r *Request) SetHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

func (r *Request) Get(url string) (*Response, error) {
	return r.Send(http.MethodGet, url)
}

func (r *Request) Post(url string) (*Response, error) {
	return r.Send(http.MethodPost, url)
}

func (r *Request) Put(url string) (*Response, error) {
	return r.Send(http.MethodPut, url)
}

func (r *Request) Delete(url string) (*Response, error) {
	return r.Send(http.MethodDelete, url)
}

// TODO: Make this accept more types ([]byte, string, ...)
func (r *Request) SetBody(v any) *Request {
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

		r.url, err = url.Parse(r.client.BaseURL + tempURL)
		if err != nil {
			return nil, err
		}
	}

	resp, err = r.Do()
	if err != nil {
		return nil, err
	} else if resp.Err != nil {
		return nil, resp.Err
	}

	return resp, nil
}

func (r *Request) Do() (resp *Response, err error) {
	_, span := r.client.tracer.Start(r.ctx, r.url.Hostname()) // TODO:
	defer span.End()

	span.SetAttributes(attribute.String("name", "daisy"))

	for {
		// TODO: Retry

		response, err := r.client.roundTrip(r)
		if err != nil {
			return nil, err
		}

		return response, nil
	}
}
