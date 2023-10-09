package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/flanksource/commons/logger"
)

const contentType = "Content-Type"

// Client is a type that represents an HTTP client
type Client struct {
	httpClient *http.Client

	// authConfig specifies the authentication configuration
	authConfig *AuthConfig

	// transportMiddlewares are like http middlewares for transport
	transportMiddlewares []Middleware

	// retryConfig specifies the configuration for retries.
	retryConfig RetryConfig

	// ConnectTo specifies the host to connect to.
	// Might be different from the host specified in the URL.
	ConnectTo string

	// headers are automatically added to all requests
	headers http.Header

	// baseURL is added as a prefix to all URLs
	baseURL string

	// Log controls whether the request response should be logged or not
	Log bool

	// GET's are on TRACE, PUT/PATCH/POST are on Debug, and DELETE are on Info
	Logger logger.Logger

	// Timeout specifies a time limit for requests made by this Client.
	//  Default: 2 minutes
	Timeout time.Duration

	// ProxyHost specifies a proxy
	ProxyHost string

	// ProxyPort specifies the proxy's port
	ProxyPort uint16

	// DNSCache specifies whether to cache DNS lookups
	DNSCache bool
}

// NewClient configures a new HTTP client using given configuration
func NewClient() *Client {
	client := &http.Client{
		Timeout: time.Minute * 2,
	}

	return &Client{
		httpClient: client,
		headers:    http.Header{},
		Logger:     logger.StandardLogger(),
	}
}

// R create a new request.
func (c *Client) R(ctx context.Context) *Request {
	return &Request{
		ctx:         ctx,
		client:      c,
		headers:     make(http.Header),
		queryParams: make(url.Values),
		retryConfig: c.retryConfig,
	}
}

// Retry configuration retrying on failure with exponential backoff.
//
// Base duration of a second & an exponent of 2 is a good option.
func (c *Client) Retry(maxRetries uint, baseDuration time.Duration, exponent float64) *Client {
	c.retryConfig.MaxRetries = maxRetries
	c.retryConfig.RetryWait = baseDuration
	c.retryConfig.Factor = exponent
	return c
}

func (c *Client) BaseURL(url string) *Client {
	c.baseURL = url
	return c
}

func (c *Client) Header(key, val string) *Client {
	c.headers.Set(key, val)
	return c
}

func (c *Client) Host(host string) *Client {
	c.ConnectTo = host
	return c
}

func (c *Client) Transport(rt http.RoundTripper) *Client {
	c.httpClient.Transport = rt
	return c
}

func (c *Client) BasicAuth(username, password string) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Username = username
	c.authConfig.Password = password
	return c
}

func (c *Client) roundTrip(r *Request) (resp *Response, err error) {
	// setup url and host
	var host string
	if r.client.ConnectTo != "" {
		host = r.client.ConnectTo
	} else if h := r.getHeader("Host"); h != "" {
		host = h // Host header override
	} else {
		host = r.url.Host
	}

	req, err := http.NewRequestWithContext(r.ctx, r.method, r.url.String(), r.body)
	if err != nil {
		return nil, err
	}

	// use the headers from the client & add/overwrite them with headers from the request
	req.Header = c.headers.Clone()
	for k, v := range r.headers.Clone() {
		for _, vv := range v {
			req.Header.Set(k, vv)
		}
	}

	queryParam := req.URL.Query()
	for k, v := range r.queryParams {
		for _, vv := range v {
			queryParam.Set(k, vv)
		}
	}
	req.URL.RawQuery = queryParam.Encode()

	req.Host = host
	if r.client.authConfig != nil {
		req.SetBasicAuth(r.client.authConfig.Username, r.client.authConfig.Password)
	}

	roundTripper := applyMiddleware(RoundTripperFunc(r.client.httpClient.Do), r.client.transportMiddlewares...)
	httpResponse, err := roundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	response := &Response{
		Response: httpResponse,
	}
	return response, nil
}

// Use adds middleware to the client that wraps the client's transport
func (c *Client) Use(middlewares ...Middleware) *Client {
	c.transportMiddlewares = append(c.transportMiddlewares, middlewares...)
	return c
}

func applyMiddleware(h http.RoundTripper, middleware ...Middleware) http.RoundTripper {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
