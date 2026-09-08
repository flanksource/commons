// Package http provides an enhanced HTTP client with built-in support for
// authentication, retries, tracing, and middleware.
//
// The client supports multiple authentication methods (Basic, Digest, NTLM, OAuth, AWS Sigv4),
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
// With AWS Sigv4 Authentication:
//
//	cfg, _ := awsconfig.LoadDefaultConfig(ctx)
//	client := http.NewClient().AWSAuthSigV4(cfg).AWSService("s3")
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
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/flanksource/commons/dns"
	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
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
	FormParams:      true,
	Headers:         true,
	ResponseHeaders: true,
	TLS:             true,
	Auth:            true,
}

var TraceHeaders = TraceConfig{
	Body:            false,
	Response:        false,
	QueryParam:      true,
	FormParams:      true,
	Headers:         true,
	ResponseHeaders: true,
	TLS:             false,
	Auth:            true,
}

func (a *AuthConfig) IsEmpty() bool {
	return a.Username == "" && a.Password == "" && a.AWSCredentialsProvider == nil
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

	// AWS Sigv4 authentication
	AWSCredentialsProvider aws.CredentialsProvider
	AWSRegion              string
	AWSService             string
	AWSEndpoint            string
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
	httpClient         *http.Client
	connectDialContext func(context.Context, string, string) (net.Conn, error)

	// authConfig specifies the authentication configuration
	authConfig *AuthConfig

	// traceConfig stores the trace configuration for use by auth middlewares
	traceConfig TraceConfig

	// transportMiddlewares are installed directly around httpClient.Transport,
	// outermost first, so they execute in the order Use registered them.
	transportMiddlewares []middlewares.Middleware

	// baseTransport is httpClient.Transport as it stood before the first Use,
	// so chained Use calls can rebuild the chain in registration order.
	baseTransport http.RoundTripper

	// authMiddlewares wrap the whole client call (httpClient.Do) outside the
	// HAR middlewares, so headers they add are captured in HAR entries.
	authMiddlewares []middlewares.Middleware

	// retryConfig specifies the configuration for retries.
	retryConfig RetryConfig

	// retryStrategy, when non-nil, fully owns the retry decision for every
	// request, superseding retryConfig. See RetryStrategy.
	retryStrategy RetryStrategy

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

	curlLog bool

	// harCollector is also attached to authentication transports that do not
	// use the client's primary transport.
	harCollector *har.Collector

	// harMiddlewares capture each transport attempt, including redirect hops,
	// after request authentication has been applied.
	harMiddlewares []middlewares.Middleware

	// maxRedirects controls how many redirects to follow. -1 means no following.
	maxRedirects int

	// logger, when non-nil, overrides logger.GetLogger() for client-internal
	// logging (currently only WithHttpLogging consumes it). Set via WithLogger.
	logger logger.Logger

	// harPath is the path WithContext attached a HAR collector for. Empty
	// when no HAR is wired. Read-only after WithContext returns; exposed
	// indirectly so a higher-level context can flush the collector.
	harPath string

	// traceMW points at the single installed trace middleware so subsequent
	// TraceToStdout calls merge into one config instead of stacking another
	// middleware. See TraceToStdout for the dedupe path.
	traceMW *traceMiddlewareHandle
}

// RoundTrip implements http.RoundTripper.
func (c *Client) RoundTrip(r *http.Request) (*http.Response, error) {
	// Convert http.Request to our custom Request type
	req := &Request{
		ctx:           r.Context(),
		client:        c,
		method:        r.Method,
		url:           r.URL,
		body:          r.Body,
		headers:       r.Header,
		queryParams:   r.URL.Query(),
		retryConfig:   c.retryConfig,
		retryStrategy: c.retryStrategy,
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
		httpClient:   client,
		userAgent:    "flanksource-commons/0",
		headers:      http.Header{},
		maxRedirects: 10,
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
		ctx:           ctx,
		client:        c,
		headers:       make(http.Header),
		queryParams:   make(url.Values),
		retryConfig:   c.retryConfig,
		retryStrategy: c.retryStrategy,
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

// RetryStrategy installs a callback that decides whether each HTTP attempt
// should be retried. When set, it fully supersedes the legacy Retry()
// exponential-backoff loop and owns the retry policy (including the
// attempt cap). See the RetryStrategy type and the RetryOnStatus helper.
//
// Example — retry on 429 and 5xx, honoring Retry-After:
//
//	client := http.NewClient().RetryStrategy(
//	    http.RetryOnStatus(5, time.Second,
//	        429, 502, 503, 504),
//	)
func (c *Client) RetryStrategy(fn RetryStrategy) *Client {
	c.retryStrategy = fn
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

func (c *Client) CurlLog() *Client {
	c.curlLog = true
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

// AWSAuthSigV4 configures AWS Signature Version 4 authentication.
// With no arguments, it loads the default AWS config (env vars, ~/.aws, IAM role, etc.).
// Pass an aws.Config to use specific credentials/region.
//
// Example with defaults:
//
//	client.AWSAuthSigV4()
//
// Example with explicit config:
//
//	cfg, _ := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("us-east-1"))
//	client.AWSAuthSigV4(cfg)
func (c *Client) AWSAuthSigV4(cfgs ...aws.Config) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}

	if len(cfgs) > 0 {
		cfg := cfgs[0]
		c.authConfig.AWSCredentialsProvider = cfg.Credentials
		c.authConfig.AWSRegion = cfg.Region
	} else {
		cfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err == nil {
			c.authConfig.AWSCredentialsProvider = cfg.Credentials
			c.authConfig.AWSRegion = cfg.Region
		}
	}
	return c
}

// AWSService sets the AWS service name for SigV4 signing.
// If not set, the service is inferred from the request URL hostname.
func (c *Client) AWSService(service string) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}
	c.authConfig.AWSService = service
	return c
}

// AWSEndpoint sets a custom AWS endpoint (e.g., for LocalStack testing).
func (c *Client) AWSEndpoint(endpoint string) *Client {
	if c.authConfig == nil {
		c.authConfig = &AuthConfig{}
	}
	c.authConfig.AWSEndpoint = endpoint
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
	if c.harCollector != nil {
		existing := config.TokenTransport
		harMiddleware := c.harCollector.Middleware()
		if existing != nil {
			config.TokenTransport = func(rt http.RoundTripper) http.RoundTripper {
				return existing(harMiddleware(rt))
			}
		} else {
			config.TokenTransport = harMiddleware
		}
	}
	// Installed outside the HAR middlewares (not via Use, which installs
	// around the transport) so HAR captures the Authorization header.
	c.authMiddlewares = append(c.authMiddlewares, middlewares.NewOauthTransport(config).RoundTripper)
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
	c.traceConfig = config
	c.Use(middlewares.NewTracedTransport(config).RoundTripper)
	return c
}

// TraceToStdout installs the stdout trace middleware. Calling it a second
// time on the same Client merges the new config into the existing one
// instead of stacking a second middleware — this lets WithLogger (the
// -v ladder) and WithContext (-P http.log=) both contribute without
// doubling every traced request.
func (c *Client) TraceToStdout(config TraceConfig, verbose ...logger.Verbose) *Client {
	if c.traceMW != nil {
		mergeTraceConfig(c.traceMW.cfg, config)
		c.traceConfig = *c.traceMW.cfg
		return c
	}
	cfg := config
	handle := &traceMiddlewareHandle{cfg: &cfg}
	var v logger.Verbose
	if len(verbose) > 0 {
		v = verbose[0]
	}
	c.traceMW = handle
	c.traceConfig = cfg
	c.Use(func(rt http.RoundTripper) http.RoundTripper {
		// Build the inner logger middleware lazily on each request so
		// merges performed after this Use() call are observed.
		return middlewares.NewLogger(*handle.cfg, v)(rt)
	})
	return c
}

// traceMiddlewareHandle holds a pointer to the live TraceConfig that the
// installed trace middleware reads on every request. Subsequent
// TraceToStdout calls mutate *cfg in place via mergeTraceConfig.
type traceMiddlewareHandle struct {
	cfg *TraceConfig
}

// mergeTraceConfig OR-folds src into dst. Bool fields become true if
// either side is true; MaxBodyLength takes the larger non-zero value;
// RedactedHeaders are unioned with case-insensitive dedup; SpanName is
// kept from dst unless dst's is empty.
func mergeTraceConfig(dst *TraceConfig, src TraceConfig) {
	dst.Body = dst.Body || src.Body
	dst.Response = dst.Response || src.Response
	dst.Headers = dst.Headers || src.Headers
	dst.ResponseHeaders = dst.ResponseHeaders || src.ResponseHeaders
	dst.QueryParam = dst.QueryParam || src.QueryParam
	dst.FormParams = dst.FormParams || src.FormParams
	dst.TLS = dst.TLS || src.TLS
	dst.Timing = dst.Timing || src.Timing
	dst.Auth = dst.Auth || src.Auth
	// A "full" access log (AccessLog && !errorsOnly) from either side wins over
	// errors-only: if any contributor wants every request logged, the merge does
	// too. Errors-only survives only when no contributor requested full logging.
	dstFull := dst.AccessLog && !dst.AccessLogErrorsOnly
	srcFull := src.AccessLog && !src.AccessLogErrorsOnly
	dst.AccessLog = dst.AccessLog || src.AccessLog
	dst.AccessLogErrorsOnly = dst.AccessLog && !dstFull && !srcFull
	if src.MaxBodyLength > dst.MaxBodyLength {
		dst.MaxBodyLength = src.MaxBodyLength
	}
	dst.RedactedHeaders = appendUnique(dst.RedactedHeaders, src.RedactedHeaders...)
	if dst.SpanName == "" {
		dst.SpanName = src.SpanName
	}
}

// appendUnique returns dst with values added that aren't already present
// (case-insensitive). Used for RedactedHeaders merging.
func appendUnique(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, v := range dst {
		seen[strings.ToLower(v)] = struct{}{}
	}
	for _, v := range values {
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, v)
	}
	return dst
}

type traceLevelOptions struct {
	baseLevel logger.LogLevel
}

// TraceLevelOption configures how TraceConfigForLogLevel maps a commons logger
// level onto the HTTP trace ladder.
type TraceLevelOption func(*traceLevelOptions)

// WithTraceBaseLevel shifts the whole HTTP trace ladder to start at level.
// With the default base of Debug, -v logs a single access line; with an Info
// base, a normal info-level logger logs that access line and -v logs headers.
func WithTraceBaseLevel(level logger.LogLevel) TraceLevelOption {
	return func(opts *traceLevelOptions) {
		opts.baseLevel = level
	}
}

// TraceConfigForLogLevel maps a logger level to a TraceConfig for the stdout
// trace middleware. The ladder is relative to a configurable base level:
//
//	base + 0: access line only
//	base + 1: query/form params + request/response headers
//	base + 2: request bodies + concise TLS
//	base + 3: response bodies
//
// The default base is Debug and can be overridden with http.log.base-level,
// HTTP_LOG_BASE_LEVEL, or WithTraceBaseLevel.
func TraceConfigForLogLevel(level logger.LogLevel, options ...TraceLevelOption) (TraceConfig, bool) {
	opts := traceLevelOptions{baseLevel: configuredTraceBaseLevel()}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}

	// One level below base (INFO when base is the default Debug) installs the
	// access-log middleware in error-only mode: logPrettyAccess emits at INFO
	// for transport errors and responses >= 400 (with body), and stays silent
	// for successes. This surfaces failing requests at -v=0 without the
	// per-request access spam that full access logging (base level) produces.
	if level < opts.baseLevel-1 {
		return TraceConfig{}, false
	}

	config := TraceConfig{
		AccessLog:           true,
		AccessLogErrorsOnly: level < opts.baseLevel,
		RedactedHeaders:     []string{"Authorization"},
	}
	if level >= opts.baseLevel+1 {
		config.QueryParam = true
		config.FormParams = true
		config.Headers = true
		config.ResponseHeaders = true
	}
	if level >= opts.baseLevel+2 {
		config.MaxBodyLength = 4096
		config.Body = true
		config.TLS = true
	}
	if level >= opts.baseLevel+3 {
		config.MaxBodyLength = 4096
		config.Response = true
	}
	return config, true
}

func configuredTraceBaseLevel() logger.LogLevel {
	level := strings.TrimSpace(os.Getenv("HTTP_LOG_BASE_LEVEL"))
	if level == "" {
		level = strings.TrimSpace(properties.String("", "http.log.base-level"))
	}
	if level == "" {
		return logger.Debug
	}
	return logger.ParseLevel(logger.GetLogger(), level)
}

// WithLogger stores l as the client's logger AND, as a side effect,
// installs a stdout trace middleware whose config is derived from
// l.GetLevel() via TraceConfigForLogLevel. This makes -v / -vv "just
// work" by passing the application's standard logger:
//
//	NewClient().WithLogger(logger.StandardLogger())
//
// The trace middleware is shared with WithContext (-P http.log=) via
// TraceToStdout's dedupe — calling both is safe and merges configs.
func (c *Client) WithLogger(l logger.Logger) *Client {
	c.logger = l
	if cfg, ok := TraceConfigForLogLevel(l.GetLevel()); ok {
		c = c.TraceToStdout(cfg)
	}
	return c
}

func (c *Client) getLogger() logger.Logger {
	if c.logger != nil {
		return c.logger
	}
	return logger.GetLogger()
}

// HARLevel selects what a HAR collector captures when attached via
// WithContext. Borrowed from duty/connection/common.go's Debug/Trace
// split — at Metadata, only request/response headers + timing are
// captured (no bodies, no body re-read cost). At Full, the standard
// collector middleware captures bodies too.
// The levels live in commons/har so the middleware, the property registry and
// this client all speak one enum; these names are kept for callers.
type HARLevel = har.Level

const (
	HARDisabled = har.Disabled
	HARMetadata = har.Metadata
	HARFull     = har.Full
)

// CommonsHTTPContext is the narrow interface a context object implements
// to drive HTTP-client configuration. xerocli.Context (and could be
// duty/context.Context) satisfies it; commons/http does not require any
// other concrete dependency from the application.
//
// HARFor returns the collector, the resolved file path, and the level.
// A nil collector signals "no HAR for this feature" (the client wires no
// HAR middleware in that case). Implementations are expected to handle
// per-path collector deduplication themselves so multiple clients
// sharing the same output file share one collector.
type CommonsHTTPContext interface {
	GetLogger() logger.Logger
	HTTPTraceConfig(feature string) (TraceConfig, bool)
	HARFor(feature string) (collector *har.Collector, path string, level HARLevel)
}

// WithContext configures the client from a context-object's data
// accessors. The feature name lets implementations distinguish callers
// (e.g. "takealot" vs "xero") for per-feature property overrides.
//
// Order of operations:
//  1. WithLogger(ctx.GetLogger()) — installs the -v ladder trace.
//  2. ctx.HTTPTraceConfig(feature) — if set, merges into the trace via
//     TraceToStdout's dedupe path. Authorization is added to
//     RedactedHeaders.
//  3. ctx.HARFor(feature) — if a collector is returned, attaches it
//     either as a full-body capture (HARFull) or as a metadata-only
//     middleware (HARMetadata).
//
// WithContext does NOT register any lifecycle hook — the context owns
// flushing (e.g. via context.AfterFunc on cancellation).
func (c *Client) WithContext(ctx CommonsHTTPContext, feature string) *Client {
	c = c.WithLogger(ctx.GetLogger())
	if cfg, ok := ctx.HTTPTraceConfig(feature); ok {
		cfg.RedactedHeaders = appendUnique(cfg.RedactedHeaders, "Authorization")
		c = c.TraceToStdout(cfg)
	}
	if collector, path, level := ctx.HARFor(feature); collector != nil {
		c.harPath = path
		switch level {
		case HARFull:
			c = c.HARCollector(collector)
		case HARMetadata:
			c.Use(har.NewMetadataMiddleware(collector.Config, collector.Add))
		}
	}
	return c
}

// WriteHARFile serializes collector.Entries() into a HAR 1.2 file at
// path. Designed for use from a context.AfterFunc hook owned by the
// caller — commons/http does not register any lifecycle itself, though
// har.Registry does it for property-driven capture.
func WriteHARFile(collector *har.Collector, path string) error {
	return har.WriteFile(collector, path)
}

// HAR enables HAR capture with default config.
// handler is called with each request/response entry after the round-trip.
// HAR(nil) is a no-op.
func (c *Client) HAR(handler func(*har.Entry)) *Client {
	return c.HARWithConfig(har.DefaultConfig(), handler)
}

// HARWithConfig enables HAR capture with a custom HARConfig.
func (c *Client) HARWithConfig(config har.HARConfig, handler func(*har.Entry)) *Client {
	c.harMiddlewares = append(c.harMiddlewares, har.NewMiddleware(config, handler))
	return c
}

// HARCollector enables HAR capture using a Collector that accumulates all
// entries (including OAuth token fetches, redirect hops, and retry attempts).
func (c *Client) HARCollector(collector *har.Collector) *Client {
	c.harCollector = collector
	c.harMiddlewares = append(c.harMiddlewares, collector.Middleware())
	return c
}

// RedirectPolicy controls redirect following. maxRedirects=0 disables
// redirect following entirely. Values >0 limit the number of redirects.
func (c *Client) RedirectPolicy(maxRedirects int) *Client {
	c.maxRedirects = maxRedirects
	return c
}

// WithHttpLogging enables HTTP request/response logging based on the provided log levels.
//
// Parameters:
//   - headerLevel: The minimum log level required to log HTTP headers (e.g., logger.Debug)
//   - bodyLevel: The minimum log level required to log request/response bodies (e.g., logger.Trace)
//
// Example:
//
//	client.WithHttpLogging(logger.Debug, logger.Trace)
//
// This will log headers when debug logging is enabled (-v or -v 1) and
// bodies when trace logging is enabled (-vv or -v 2 or higher).
//
// Note: When using with cobra commands, ensure UseCobraFlags is called
// in PersistentPreRun to properly parse -v N syntax.
func (c *Client) WithHttpLogging(headerLevel, bodyLevel logger.LogLevel) *Client {
	c.Use(func(rt http.RoundTripper) http.RoundTripper {
		return logger.NewHttpLoggerWithLevels(c.getLogger(), rt, headerLevel, bodyLevel)
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

	if len(r.queryParams) > 0 {
		raw := req.URL.RawQuery
		for k, v := range r.queryParams {
			for _, vv := range v {
				if raw != "" {
					raw += "&"
				}
				raw += url.QueryEscape(k) + "=" + url.QueryEscape(vv)
			}
		}
		req.URL.RawQuery = raw
	}
	// Set basic auth only if not using AWS Sigv4
	if r.client.authConfig != nil && !r.client.authConfig.IsEmpty() && r.client.authConfig.AWSCredentialsProvider == nil {
		req.SetBasicAuth(r.client.authConfig.Username, r.client.authConfig.Password)
	}

	if c.proxyURL != "" {
		proxyURL, err := url.Parse(c.proxyURL)
		if err != nil {
			return nil, err
		}

		c.setProxy(proxyURL)
	}

	if c.curlLog {
		base := r.client.httpClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		r.client.httpClient.Transport = &curlLogTransport{base: base}
	}

	transport := r.client.httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	transport = applyMiddleware(transport, r.client.harMiddlewares...)

	if c.authConfig != nil {
		if c.authConfig.AWSCredentialsProvider != nil {
			awsCfg := middlewares.AWSSigv4Config{
				Region:              c.authConfig.AWSRegion,
				Service:             c.authConfig.AWSService,
				Endpoint:            c.authConfig.AWSEndpoint,
				CredentialsProvider: c.authConfig.AWSCredentialsProvider,
			}
			if c.traceConfig.Auth {
				awsCfg.Tracer = func(msg string) { logger.Tracef("%s", msg) }
			}
			transport = middlewares.NewAWSSigv4Transport(awsCfg, transport)
		} else {
			parts := strings.Split(c.authConfig.Username, "@")
			domain := ""
			if len(parts) > 1 {
				domain = parts[1]
			}

			if c.authConfig.Ntlmv2 {
				transport = &httpntlmv2.NtlmTransport{
					Domain:       domain,
					User:         parts[0],
					Password:     c.authConfig.Password,
					RoundTripper: transport,
				}
			} else if c.authConfig.Ntlm {
				transport = &httpntlm.NtlmTransport{
					Domain:   domain,
					User:     parts[0],
					Password: c.authConfig.Password,
				}
				// The legacy NTLM transport does not accept an underlying transport.
				transport = applyMiddleware(transport, r.client.harMiddlewares...)
			} else if c.authConfig.Digest {
				transport = newDigestTransport(c.authConfig.Username, c.authConfig.Password, transport)
			}
		}
	}

	httpClient := *r.client.httpClient
	httpClient.Transport = transport
	httpClient.CheckRedirect = c.checkRedirectFunc()

	roundTripper := applyMiddleware(middlewares.RoundTripperFunc(httpClient.Do), r.client.authMiddlewares...)
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
func (c *Client) Use(middleware ...middlewares.Middleware) *Client {
	if c.httpClient.Transport == nil {
		c.InsecureSkipVerify(true)
	}
	// The whole chain is rebuilt from the transport Use first saw, so a later
	// Use appends behind the middleware already registered instead of wrapping
	// outside it: Use(A).Use(B) and Use(A, B) both run A before B.
	if len(c.transportMiddlewares) == 0 {
		c.baseTransport = c.httpClient.Transport
	}
	c.transportMiddlewares = append(c.transportMiddlewares, middleware...)
	c.httpClient.Transport = applyMiddleware(c.baseTransport, c.transportMiddlewares...)
	return c
}

func (c *Client) checkRedirectFunc() func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if c.maxRedirects == 0 {
			return http.ErrUseLastResponse
		}
		if len(via) >= c.maxRedirects {
			return fmt.Errorf("stopped after %d redirects", c.maxRedirects)
		}

		return nil
	}
}

func applyMiddleware(h http.RoundTripper, middleware ...middlewares.Middleware) http.RoundTripper {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
