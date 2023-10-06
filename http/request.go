package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
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

// Header set a header for the request.
func (r *Request) Header(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

func (r *Request) Get(ctx context.Context, url string) (*Response, error) {
	return r.Send(ctx, http.MethodGet, url)
}

func (r *Request) Post(ctx context.Context, url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(ctx, http.MethodPost, url)
}

func (r *Request) Put(ctx context.Context, url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(ctx, http.MethodPut, url)
}

func (r *Request) Patch(ctx context.Context, url string, body any) (*Response, error) {
	r.setBody(body)
	return r.Send(ctx, http.MethodPatch, url)
}

func (r *Request) Delete(ctx context.Context, url string) (*Response, error) {
	return r.Send(ctx, http.MethodDelete, url)
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

func (r *Request) Send(ctx context.Context, method, reqURL string) (resp *Response, err error) {
	r.ctx = ctx
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
	} else if resp.Err != nil {
		return nil, resp.Err
	}

	return resp, nil
}

func (r *Request) Do() (resp *Response, err error) {
	for {
		// TODO: Retry

		response, err := r.client.roundTrip(r)
		if err != nil {
			return nil, err
		}

		return response, nil
	}
}
