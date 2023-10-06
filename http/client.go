package http

import (
	"net/http"
	"time"

	"github.com/flanksource/commons/logger"
	"go.opentelemetry.io/otel/trace"
)

const contentType = "Content-Type"

var contentTypesToLog = []string{
	"text",
	"json",
	"yml",
}

// Client is a type that represents an HTTP client
type Client struct {
	httpClient *http.Client

	tracer trace.Tracer

	// Auth specifies the authentication configuration
	Auth *AuthConfig

	// Transport specifies the transport configuration
	Transport *TransportConfig

	// Retries specifies the configuration for retries.
	Retries *RetryConfig

	// ConnectTo specifies the host to connect to.
	// Might be different from the host specified in the URL.
	ConnectTo string

	// Headers are automatically added to all requests
	Headers http.Header

	// BaseURL is added as a prefix to all URLs
	BaseURL string

	// Specify if response body should be logged
	TraceBody bool

	// TraceResponse controls if the response needs to be traced.
	// This doesn't include the response body.
	TraceResponse bool

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
		Headers:    http.Header{},
		Logger:     logger.StandardLogger(),
	}
}

// R create a new request.
func (c *Client) R() *Request {
	return &Request{
		client:      c,
		headers:     make(http.Header),
		retryConfig: c.Retries,
	}
}

func (c *Client) SetBaseURL(url string) *Client {
	c.BaseURL = url
	return c
}

func (c *Client) SetHeader(key, val string) *Client {
	c.Headers.Set(key, val)
	return c
}

func (c *Client) SetHost(host string) *Client {
	c.ConnectTo = host
	return c
}

func (c *Client) SetTracer(tracer trace.Tracer) *Client {
	c.tracer = tracer
	return c
}

func (c *Client) SetBasicAuth(username, password string) *Client {
	if c.Auth == nil {
		c.Auth = &AuthConfig{}
	}

	c.Auth.Username = username
	c.Auth.Password = password
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
	req.Header = r.headers.Clone()
	req.Host = host

	httpResponse, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	response := &Response{
		Response: httpResponse,
	}
	return response, nil
}
