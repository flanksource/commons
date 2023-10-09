package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

const contentType = "Content-Type"

type Middleware func(http.RoundTripper) http.RoundTripper

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type AuthConfig struct {
	// Username for basic Auth
	Username string

	// Password for basic Auth
	Password string

	// Ntlm controls whether to use NTLM
	Ntlm bool

	// Ntlmv2 controls whether to use NTLMv2
	Ntlmv2 bool
}

// Client is a type that represents an HTTP client
type Client struct {
	httpClient *http.Client

	// authConfig specifies the authentication configuration
	authConfig *AuthConfig

	// transportMiddlewares are like http middlewares for transport
	transportMiddlewares []Middleware

	// retryConfig specifies the configuration for retries.
	retryConfig RetryConfig

	// connectTo specifies the host to connect to.
	// Might be different from the host specified in the URL.
	connectTo string

	// headers are automatically added to all requests
	headers http.Header

	// baseURL is added as a prefix to all URLs
	baseURL string

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

// ConnectTo specifies the host:port on which the URL is sought.
// If empty, the URL's host is used.
func (c *Client) ConnectTo(host string) *Client {
	c.connectTo = host
	return c
}

// Timeout specifies a time limit for requests made by this Client.
//
//	Default: 2 minutes
func (c *Client) Timeout(d time.Duration) *Client {
	c.httpClient.Timeout = d
	return c
}

// DisableKeepAlives prevents reuse of TCP connections
func (c *Client) DisableKeepAlive(val bool) *Client {
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport
	}

	customTransport := c.httpClient.Transport.(*http.Transport).Clone()
	customTransport.DisableKeepAlives = val
	c.httpClient.Transport = customTransport
	return c
}

// InsecureSkipVerify controls whether a client verifies the server's
// certificate chain and host name
func (c *Client) InsecureSkipVerify(val bool) *Client {
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport
	}

	customTransport := c.httpClient.Transport.(*http.Transport).Clone()

	if customTransport.TLSClientConfig == nil {
		customTransport.TLSClientConfig = &tls.Config{}
	}

	customTransport.TLSClientConfig.InsecureSkipVerify = val
	c.httpClient.Transport = customTransport
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
	if r.client.connectTo != "" {
		host = r.client.connectTo
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
