// Package http provides an enhanced HTTP client with built-in support for
// authentication, retries, tracing, and middleware.
//
// The client supports multiple authentication methods (Basic, Digest, NTLM, OAuth),
// automatic retries with exponential backoff, request/response tracing, and a
// flexible middleware system.
//
// Basic Usage:
//
//	client := http.NewClient()
//	resp, err := client.R(context.Background()).GET("https://api.example.com/data")
//
// With Authentication:
//
//	client := http.NewClient().
//		Auth("username", "password").
//		Digest(true)
//
// With Retries and Timeout:
//
//	client := http.NewClient().
//		Retry(3, time.Second, 2.0).
//		Timeout(30 * time.Second)
//
// With Request Tracing:
//
//	client := http.NewClient().
//		Trace(http.TraceAll)  // Log full request/response details
//
// The client can also be used as a standard http.RoundTripper:
//
//	httpClient := &http.Client{
//		Transport: http.NewClient().Auth("user", "pass").RoundTripper(),
//	}
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
	"github.com/flanksource/commons/logger"
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

// Client is an enhanced HTTP client with built-in support for authentication,
// retries, tracing, and middleware. It provides a fluent API for configuring
// requests and can be used as a drop-in replacement for http.Client.
//
// Example:
//
//	client := http.NewClient().
//		Auth("user", "pass").
//		Retry(3, time.Second, 2.0).
//		Trace(http.TraceHeaders).
//		Timeout(30 * time.Second)
//
//	resp, err := client.R(ctx).
//		GET("https://api.example.com/data")
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

// RoundTrip implements http.RoundTripper.
func (c *Client) RoundTrip(r *http.Request) (*http.Response, error) {
	// Convert http.Request to our custom Request type
	req := &Request{
		ctx:         r.Context(),
		client:      c,
		method:      r.Method,
		url:         r.URL,
		body:        r.Body,
		headers:     r.Header,
		queryParams: r.URL.Query(),
		retryConfig: c.retryConfig,
	}

	resp, err := c.roundTrip(req)
	if resp == nil {
		return nil, err
	}
	return resp.Response, err
}

// NewClient creates a new HTTP client with default settings.
// The client has a default timeout of 2 minutes and can be customized
// using the fluent API methods.
//
// Example:
//
//	client := http.NewClient().
//		BaseURL("https://api.example.com").
//		Header("X-API-Key", "secret").
//		InsecureSkipVerify(true)
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

// R creates a new request with the given context.
// The request inherits all settings from the client (headers, auth, etc.)
// and can be further customized before execution.
//
// Example:
//
//	resp, err := client.R(ctx).
//		Header("X-Request-ID", "123").
//		QueryParam("page", "1").
//		GET("/users")
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

// Retry configures automatic retry behavior with exponential backoff.
// Failed requests will be retried up to maxRetries times, with delays
// calculated as baseDuration * (exponent ^ attemptNumber).
//
// Parameters:
//   - maxRetries: Maximum number of retry attempts (0 disables retries)
//   - baseDuration: Initial delay between retries
//   - exponent: Multiplier for exponential backoff (typically 2.0)
//
// Example:
//
//	// Retry up to 3 times with delays of 1s, 2s, 4s
//	client.Retry(3, time.Second, 2.0)
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

// ConnectTo overrides the target host:port for requests.
// This is useful for testing against specific IPs or when dealing with
// DNS issues. The original hostname is preserved in the Host header.
//
// Example:
//
//	// Connect to a specific IP instead of using DNS
//	client.ConnectTo("192.168.1.100:8080")
//
//	// Test against localhost while preserving the original Host header
//	client.ConnectTo("localhost:3000")
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

// TLSConfig configures advanced TLS settings including custom CAs,
// client certificates, and handshake timeout.
//
// Example:
//
//	client.TLSConfig(TLSConfig{
//		CA:                 caPEM,        // Custom CA certificate
//		Cert:               clientCert,   // Client certificate for mTLS
//		Key:                clientKey,    // Client private key
//		InsecureSkipVerify: false,        // Verify server certificate
//		HandshakeTimeout:   10 * time.Second,
//	})
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

// InsecureSkipVerify disables TLS certificate verification.
// WARNING: This makes the client vulnerable to man-in-the-middle attacks.
// Only use in development or when connecting to services with self-signed certificates.
//
// Example:
//
//	// Accept any certificate (dangerous!)
//	client.InsecureSkipVerify(true)
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

// Auth configures authentication credentials for the client.
// By default, this sets up HTTP Basic Authentication.
// Use Digest(), NTLM(), or NTLMV2() to switch authentication methods.
//
// Example:
//
//	// Basic auth
//	client.Auth("username", "password")
//
//	// Digest auth
//	client.Auth("username", "password").Digest(true)
//
//	// NTLM auth with domain
//	client.Auth("DOMAIN\\username", "password").NTLM(true)
func (c *Client) Auth(username, password string) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Username = username
	c.authConfig.Password = password
	return c
}

// OAuth configures OAuth 2.0 authentication for the client.
// Supports various OAuth flows including client credentials and authorization code.
//
// Example:
//
//	client.OAuth(middlewares.OauthConfig{
//		ClientID:     "client-id",
//		ClientSecret: "client-secret",
//		TokenURL:     "https://auth.example.com/token",
//		Scopes:       []string{"read", "write"},
//	})
func (c *Client) OAuth(config middlewares.OauthConfig) *Client {
	c.Use(middlewares.NewOauthTransport(config).RoundTripper)
	return c
}

// Trace enables request/response tracing with customizable detail levels.
// Traced information is added to the context and can be retrieved by middleware.
//
// Use predefined configs for common scenarios:
//   - TraceAll: Full details including headers, body, and TLS
//   - TraceHeaders: Headers and query params only (no body)
//
// Example:
//
//	client.Trace(http.TraceConfig{
//		Headers:         true,
//		ResponseHeaders: true,
//		Body:            true,
//		Response:        true,
//		MaxBodyLength:   1024,  // Limit body size in traces
//	})
func (c *Client) Trace(config TraceConfig) *Client {
	c.Use(middlewares.NewTracedTransport(config).RoundTripper)
	return c
}

func (c *Client) TraceToStdout(config TraceConfig) *Client {
	c.Use(middlewares.NewLogger(config))
	return c
}

// WithHttpLogging enables HTTP request/response logging based on the provided log levels.
// 
// Parameters:
//   - headerLevel: The minimum log level required to log HTTP headers (e.g., logger.Debug)
//   - bodyLevel: The minimum log level required to log request/response bodies (e.g., logger.Trace)
//
// Example:
//   client.WithHttpLogging(logger.Debug, logger.Trace)
//   
// This will log headers when debug logging is enabled (-v or -v 1) and 
// bodies when trace logging is enabled (-vv or -v 2 or higher).
//
// Note: When using with cobra commands, ensure UseCobraFlags is called
// in PersistentPreRun to properly parse -v N syntax.
func (c *Client) WithHttpLogging(headerLevel, bodyLevel logger.LogLevel) *Client {
	c.Use(func(rt http.RoundTripper) http.RoundTripper {
		return logger.NewHttpLoggerWithLevels(logger.GetLogger(), rt, headerLevel, bodyLevel)
	})
	return c
}

func (c *Client) Digest(val bool) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	c.authConfig.Digest = val
	return c
}

func (c *Client) RoundTripper() http.RoundTripper {
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

// Use adds middleware to the client's transport chain.
// Middleware functions wrap the underlying RoundTripper and can modify
// requests/responses, add logging, implement caching, etc.
//
// Middleware is applied in the order it was added.
//
// Example:
//
//	client.Use(func(rt http.RoundTripper) http.RoundTripper {
//		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
//			req.Header.Set("X-Custom", "value")
//			return rt.RoundTrip(req)
//		})
//	})
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
