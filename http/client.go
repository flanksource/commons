package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	dac "github.com/Snawoot/go-http-digest-auth-client"
	"github.com/flanksource/commons/dns"
	"github.com/flanksource/commons/http/middlewares"
	httpntlm "github.com/vadimi/go-http-ntlm"
	httpntlmv2 "github.com/vadimi/go-http-ntlm/v2"
)

type TraceConfig = middlewares.TraceConfig

type OauthConfig = middlewares.OauthConfig

var AuthStyleInHeader = middlewares.AuthStyleInHeader
var AuthStyleInParams = middlewares.AuthStyleInParams
var AuthStyleAutoDetect = middlewares.AuthStyleAutoDetect

var TraceAll = TraceConfig{
	MaxBodyLength:   4096,
	Body:            true,
	Response:        true,
	QueryParam:      true,
	Headers:         true,
	ResponseHeaders: true,
	TLS:             true,
}

var TraceHeaders = TraceConfig{
	Body:            false,
	Response:        false,
	QueryParam:      true,
	Headers:         true,
	ResponseHeaders: true,
	TLS:             false,
}

func (a *AuthConfig) IsEmpty() bool {
	return a.Username == "" && a.Password == ""
}

type AuthConfig struct {
	// Username for basic Auth
	Username string

	// Password for basic Auth
	Password string

	// Use digest access authentication
	Digest bool

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
	transportMiddlewares []middlewares.Middleware

	// retryConfig specifies the configuration for retries.
	retryConfig RetryConfig

	// connectTo specifies the host to connect to.
	// Might be different from the host specified in the URL.
	connectTo string

	// headers are automatically added to all requests
	headers http.Header

	// baseURL is added as a prefix to all URLs
	baseURL string

	// proxyURL is the url to use as a proxy
	proxyURL string

	// cacheDNS specifies whether to cache DNS lookups
	cacheDNS bool

	userAgent string

	tlsConfig *tls.Config
}

// NewClient configures a new HTTP client using given configuration
func NewClient() *Client {
	client := &http.Client{
		Timeout: time.Minute * 2,
	}

	return &Client{
		httpClient: client,
		userAgent:  "flanksource-commons/0",
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

func (c *Client) UserAgent(agent string) *Client {
	c.userAgent = agent
	return c
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

func (c *Client) CacheDNS(val bool) *Client {
	c.cacheDNS = val
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

func (c *Client) initTLSConfig() {
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport
	}

	customTransport := c.httpClient.Transport.(*http.Transport).Clone()
	if customTransport.TLSClientConfig == nil {
		customTransport.TLSClientConfig = &tls.Config{}
	}
}

type TLSConfig struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name
	InsecureSkipVerify bool
	// HandshakeTimeout defaults to 10 seconds
	HandshakeTimeout time.Duration
	// PEM encoded certificate of the CA to verify the server certificate
	CA string
	// PEM encoded client certificate
	Cert string
	// PEM encoded client private key
	Key string
}

func (c *Client) TLSConfig(conf TLSConfig) (*Client, error) {
	c.initTLSConfig()

	if conf.HandshakeTimeout == 0 {
		conf.HandshakeTimeout = time.Second * 10
	}

	transport := c.httpClient.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig.InsecureSkipVerify = conf.InsecureSkipVerify
	transport.TLSHandshakeTimeout = conf.HandshakeTimeout

	if conf.CA != "" {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}

		if !certPool.AppendCertsFromPEM([]byte(conf.CA)) {
			return nil, fmt.Errorf("failed to append ca certificate")
		}
		transport.TLSClientConfig.RootCAs = certPool
	}

	if conf.Cert != "" && conf.Key != "" {
		cert, err := tls.X509KeyPair([]byte(conf.Cert), []byte(conf.Key))
		if err != nil {
			return nil, fmt.Errorf("failed to create client certificate: %v", err)
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	c.tlsConfig = transport.TLSClientConfig
	c.httpClient.Transport = transport
	return c, nil
}

// InsecureSkipVerify controls whether a client verifies the server's
// certificate chain and host name
func (c *Client) InsecureSkipVerify(val bool) *Client {
	c.initTLSConfig()

	customTransport := c.httpClient.Transport.(*http.Transport).Clone()
	customTransport.TLSClientConfig.InsecureSkipVerify = val
	c.tlsConfig = customTransport.TLSClientConfig
	c.httpClient.Transport = customTransport
	return c
}

func (c *Client) Proxy(url string) *Client {
	c.proxyURL = url
	return c
}

func (c *Client) setProxy(proxyURL *url.URL) {
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport
	}

	customTransport := c.httpClient.Transport.(*http.Transport).Clone()
	customTransport.Proxy = http.ProxyURL(proxyURL)
	c.httpClient.Transport = customTransport
}

// Auth sets up the username & password for basic auth or NTLM.
func (c *Client) Auth(username, password string) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Username = username
	c.authConfig.Password = password
	return c
}

func (c *Client) OAuth(config middlewares.OauthConfig) *Client {
	c.Use(middlewares.NewOauthTransport(config).RoundTripper)
	return c
}

func (c *Client) Trace(config TraceConfig) *Client {
	c.Use(middlewares.NewTracedTransport(config).RoundTripper)
	return c
}

func (c *Client) TraceToStdout(config TraceConfig) *Client {
	c.Use(middlewares.NewLogger(config))
	return c
}

func (c *Client) Digest(val bool) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Digest = val
	return c
}

func (c *Client) NTLM(val bool) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Ntlm = val
	return c
}

func (c *Client) NTLMV2(val bool) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Ntlmv2 = val
	return c
}

func (c *Client) roundTrip(r *Request) (resp *Response, err error) {
	var host string
	if r.client.connectTo != "" {
		host = r.client.connectTo
	} else {
		host = r.url.Hostname()
	}

	if c.cacheDNS {
		if ips, _ := dns.CacheLookup("A", host); len(ips) > 0 {
			host = ips[0].String()
			if c.headers.Get("Host") == "" && r.headers.Get("Host") == "" {
				// add hostname back as a header
				r = r.Header("Host", r.url.Hostname())
			}
		}
	}

	if r.url.Scheme == "https" && c.tlsConfig == nil {
		// initialize default TLS settings
		c.InsecureSkipVerify(true)
	}

	uri := *r.url
	uri.Host = fmt.Sprintf("%s:%s", host, uri.Port())
	req, err := http.NewRequestWithContext(r.ctx, r.method, uri.String(), r.body)
	if err != nil {
		return nil, err
	}

	// use the headers from the client
	req.Header = c.headers.Clone()
	// add/overwrite them with headers from the request
	for k, v := range r.headers.Clone() {
		for _, vv := range v {
			req.Header.Set(k, vv)
		}
	}

	if h := req.Header.Get("Host"); h != "" {
		req.Host = h
		if c.tlsConfig != nil {
			c.tlsConfig.ServerName = h
		}
		req.Header.Del("Host")
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	queryParam := req.URL.Query()
	for k, v := range r.queryParams {
		for _, vv := range v {
			queryParam.Set(k, vv)
		}
	}
	req.URL.RawQuery = queryParam.Encode()
	if r.client.authConfig != nil && !r.client.authConfig.IsEmpty() {
		req.SetBasicAuth(r.client.authConfig.Username, r.client.authConfig.Password)
	}

	if c.proxyURL != "" {
		proxyURL, err := url.Parse(c.proxyURL)
		if err != nil {
			return nil, err
		}

		c.setProxy(proxyURL)
	}

	if c.authConfig != nil {
		parts := strings.Split(c.authConfig.Username, "@")
		domain := ""
		if len(parts) > 1 {
			domain = parts[1]
		}

		if c.authConfig.Ntlmv2 {
			r.client.httpClient.Transport = &httpntlmv2.NtlmTransport{
				Domain:       domain,
				User:         parts[0],
				Password:     c.authConfig.Password,
				RoundTripper: r.client.httpClient.Transport,
			}
		} else if c.authConfig.Ntlm {
			r.client.httpClient.Transport = &httpntlm.NtlmTransport{
				Domain:   domain,
				User:     parts[0],
				Password: c.authConfig.Password,
			}
		} else if c.authConfig.Digest {
			r.client.httpClient.Transport = dac.NewDigestTransport(c.authConfig.Username, c.authConfig.Password, r.client.httpClient.Transport)
		}
	}

	roundTripper := applyMiddleware(middlewares.RoundTripperFunc(r.client.httpClient.Do), r.client.transportMiddlewares...)
	httpResponse, err := roundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	response := &Response{
		Response: httpResponse,
	}
	return response, nil
}

func toMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range h {
		m[k] = strings.Join(v, " ")
	}
	return m
}

// Use adds middleware to the client that wraps the client's transport
func (c *Client) Use(middlewares ...middlewares.Middleware) *Client {
	c.transportMiddlewares = append(c.transportMiddlewares, middlewares...)
	return c
}

func applyMiddleware(h http.RoundTripper, middleware ...middlewares.Middleware) http.RoundTripper {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
