package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/cmd/hx/output"
	"github.com/flanksource/commons/cmd/hx/parse"
	"github.com/flanksource/commons/har"
	commonshttp "github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var rootCmd = &cobra.Command{
	Use:   "hx [METHOD] URL [ITEMS...]",
	Short: "HTTP client with httpie-style arguments",
	Long: `hx is an HTTP client that wraps flanksource/commons/http.

Positional arguments:
  METHOD       HTTP method (optional, defaults to GET or POST if body given)
  URL          Request URL (required)
  ITEMS        key=value (JSON), key:=json, key==param, Header:Value

Examples:
  hx https://httpbin.org/get
  hx https://httpbin.org/post name=test count:=42
  hx PUT https://httpbin.org/put name=updated
  hx -u user:pass https://httpbin.org/basic-auth/user/pass`,
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	SilenceErrors: true,
	RunE:         run,
}

type httpStatusError struct {
	code   int
	status string
}

func (e *httpStatusError) Error() string {
	return e.status
}

// flags
var (
	flagMethod         string
	flagData           string
	flagForm           []string
	flagHeaders        []string
	flagUser           string
	flagDigest         bool
	flagNTLM           bool
	flagToken          string
	flagAWSSigV4       bool
	flagAWSRegion      string
	flagAWSService     string
	flagAWSEndpoint    string
	flagOAuthClientID  string
	flagOAuthSecret    string
	flagOAuthTokenURL  string
	flagOAuthScopes    []string
	flagInsecure       bool
	flagCACert         string
	flagCert           string
	flagKey            string
	flagProxy          string
	flagConnectTo      string
	flagTimeout        time.Duration
	flagRetry          uint
	flagRetryWait      time.Duration
	flagRetryFactor    float64
	flagMaxRedirects   int
	flagNoFollow       bool
	flagVerbose        int
	flagHeadersOnly    bool
	flagQuiet          bool
	flagRaw            bool
	flagUserAgent      string
	flagHAROutput      string
)

func init() {
	f := rootCmd.Flags()

	f.StringVarP(&flagMethod, "method", "X", "", "HTTP method override")
	f.StringVarP(&flagData, "data", "d", "", "Request body (string or @file)")
	f.StringSliceVarP(&flagForm, "form", "f", nil, "Form data (key=value)")
	f.StringSliceVarP(&flagHeaders, "header", "H", nil, "Custom header (Key: Value)")
	f.StringVar(&flagUserAgent, "user-agent", "hx/0.1", "User-Agent header")

	f.StringVarP(&flagUser, "user", "u", "", "Basic auth (user:pass)")
	f.BoolVar(&flagDigest, "digest", false, "Use Digest auth")
	f.BoolVar(&flagNTLM, "ntlm", false, "Use NTLM auth")
	f.StringVar(&flagToken, "token", "", "Bearer token")
	f.BoolVar(&flagAWSSigV4, "aws-sigv4", false, "Enable AWS SigV4 signing (uses standard AWS credential chain)")
	f.StringVar(&flagAWSRegion, "aws-region", "", "AWS region (default: from AWS config)")
	f.StringVar(&flagAWSService, "aws-service", "", "AWS service name")
	f.StringVar(&flagAWSEndpoint, "aws-endpoint", "", "Custom AWS endpoint")
	f.StringVar(&flagOAuthClientID, "oauth-client-id", "", "OAuth2 client ID")
	f.StringVar(&flagOAuthSecret, "oauth-client-secret", "", "OAuth2 client secret")
	f.StringVar(&flagOAuthTokenURL, "oauth-token-url", "", "OAuth2 token URL")
	f.StringSliceVar(&flagOAuthScopes, "oauth-scope", nil, "OAuth2 scopes")

	f.BoolVarP(&flagInsecure, "insecure", "k", false, "Skip TLS verification")
	f.StringVar(&flagCACert, "cacert", "", "CA certificate file")
	f.StringVar(&flagCert, "cert", "", "Client certificate file")
	f.StringVar(&flagKey, "key", "", "Client key file")
	f.StringVar(&flagProxy, "proxy", "", "Proxy URL")
	f.StringVar(&flagConnectTo, "connect-to", "", "Override target host:port")

	f.DurationVar(&flagTimeout, "timeout", 2*time.Minute, "Request timeout")
	f.UintVar(&flagRetry, "retry", 0, "Max retries")
	f.DurationVar(&flagRetryWait, "retry-wait", time.Second, "Base retry delay")
	f.Float64Var(&flagRetryFactor, "retry-factor", 2.0, "Backoff multiplier")
	f.IntVar(&flagMaxRedirects, "max-redirects", 10, "Max redirects to follow")
	f.BoolVar(&flagNoFollow, "no-follow", false, "Disable redirects")

	f.CountVarP(&flagVerbose, "verbose", "v", "Verbosity (-v, -vv, -vvv)")
	f.BoolVar(&flagHeadersOnly, "headers", false, "Show headers only")
	f.BoolVarP(&flagQuiet, "quiet", "q", false, "Body only, no decoration")
	f.BoolVar(&flagRaw, "raw", false, "No colors")
	f.StringVar(&flagHAROutput, "har", "", "Write HAR capture to file (use - for stdout)")
}

func run(cmd *cobra.Command, args []string) error {
	parsed, err := parse.PositionalArgs(args)
	if err != nil {
		return err
	}

	items, err := parse.Items(parsed.Items)
	if err != nil {
		return err
	}

	for k, v := range items.Headers {
		flagHeaders = append(flagHeaders, fmt.Sprintf("%s: %s", k, v))
	}

	body, contentType, err := resolveBody(items)
	if err != nil {
		return err
	}

	hasBody := body != nil
	method := parsed.EffectiveMethod(hasBody, flagMethod)

	client, collector := buildClient()
	req := client.R(context.Background())

	for _, h := range flagHeaders {
		if k, v, ok := strings.Cut(h, ":"); ok {
			req = req.Header(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}

	if contentType != "" {
		req = req.Header("Content-Type", contentType)
	}

	for k, v := range items.QueryParams {
		req = req.QueryParam(k, v)
	}

	if flagToken != "" {
		req = req.Header("Authorization", "Bearer "+flagToken)
	}


	opts := outputOpts()

	// Print request in verbose mode
	if flagVerbose >= 1 {
		var bodyBytes []byte
		if body != nil {
			bodyBytes, _ = io.ReadAll(body)
			body = bytes.NewReader(bodyBytes)
		}
		printRequestVerbose(method, parsed.URL, req, bodyBytes, opts)
	}

	if body != nil {
		if err := req.Body(body); err != nil {
			return fmt.Errorf("setting body: %w", err)
		}
	}

	resp, err := req.Do(method, parsed.URL)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if err := output.PrintResponse(resp.Response, respBody, opts); err != nil {
		return err
	}

	if collector != nil {
		if err := writeHAR(collector.Entries(), flagHAROutput); err != nil {
			fmt.Fprintf(os.Stderr, "warning: HAR write failed: %v\n", err)
		}
	}

	if resp.StatusCode >= 400 {
		return &httpStatusError{code: resp.StatusCode, status: resp.Status}
	}
	return nil
}

func writeHAR(entries []har.Entry, dest string) error {
	file := har.File{
		Log: har.Log{
			Version: "1.2",
			Creator: har.Creator{Name: "hx", Version: version},
			Pages:   []har.Page{},
			Entries: entries,
		},
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if dest == "-" {
		_, err = fmt.Fprintf(os.Stdout, "%s\n", data)
		return err
	}
	return os.WriteFile(dest, append(data, '\n'), 0o644)
}

func outputOpts() output.Options {
	opts := output.Options{Verbosity: flagVerbose}
	switch {
	case flagQuiet:
		opts.Mode = output.ModeQuiet
	case flagHeadersOnly:
		opts.Mode = output.ModeHeaders
	case flagRaw:
		opts.Mode = output.ModeRaw
	}
	return opts
}

func printRequestVerbose(method, rawURL string, req *commonshttp.Request, body []byte, opts output.Options) {
	u, _ := url.Parse(rawURL)
	stdReq := &http.Request{
		Method: method,
		URL:    u,
		Header: make(http.Header),
		Proto:  "HTTP/1.1",
		Host:   u.Host,
	}
	for k, v := range req.GetHeaders() {
		stdReq.Header.Set(k, v)
	}
	if stdReq.Header.Get("User-Agent") == "" {
		stdReq.Header.Set("User-Agent", flagUserAgent)
	}
	if stdReq.Header.Get("Accept") == "" {
		stdReq.Header.Set("Accept", "*/*")
	}
	output.PrintRequest(stdReq, body, opts)
}

func buildClient() (*commonshttp.Client, *har.Collector) {
	var tracer func(string)
	if flagVerbose >= 1 {
		useColor := term.IsTerminal(int(os.Stderr.Fd()))
		tracer = func(msg string) {
			if useColor {
				fmt.Fprintln(os.Stderr, api.Text{Content: msg, Style: "text-yellow-400"}.ANSI())
			} else {
				fmt.Fprintln(os.Stderr, msg)
			}
		}
	}

	client := commonshttp.NewClient().
		Timeout(flagTimeout).
		UserAgent(flagUserAgent)

	var collector *har.Collector
	if flagHAROutput != "" {
		collector = har.NewCollector(har.DefaultConfig())
		client = client.HARCollector(collector)
	}

	if flagUser != "" {
		user, pass, _ := strings.Cut(flagUser, ":")
		client = client.Auth(user, pass)
		if flagDigest {
			client = client.Digest(true)
		}
		if flagNTLM {
			client = client.NTLM(true)
		}
	}

	if flagAWSSigV4 {
		var opts []func(*awsconfig.LoadOptions) error
		if flagAWSRegion != "" {
			opts = append(opts, awsconfig.WithRegion(flagAWSRegion))
		}
		cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading AWS config: %v\n", err)
			os.Exit(1)
		}
		creds, err := cfg.Credentials.Retrieve(context.Background())
		if err != nil || !creds.HasKeys() {
			fmt.Fprintf(os.Stderr, "no AWS credentials found — configure via environment, ~/.aws/credentials, or IAM role\n")
			os.Exit(1)
		}
		if cfg.Region == "" {
			fmt.Fprintf(os.Stderr, "no AWS region found — set via --aws-region, AWS_REGION, or ~/.aws/config\n")
			os.Exit(1)
		}
		client = client.AWSAuthSigV4(cfg)
		if flagAWSService != "" {
			client = client.AWSService(flagAWSService)
		}
		if flagAWSEndpoint != "" {
			client = client.AWSEndpoint(flagAWSEndpoint)
		}
	}

	if flagOAuthClientID != "" {
		client = client.OAuth(middlewares.OauthConfig{
			ClientID:     flagOAuthClientID,
			ClientSecret: flagOAuthSecret,
			TokenURL:     flagOAuthTokenURL,
			Scopes:       flagOAuthScopes,
			Tracer:       tracer,
		})
	}

	if flagInsecure {
		client = client.InsecureSkipVerify(true)
	}

	if flagCACert != "" || flagCert != "" {
		tlsCfg := commonshttp.TLSConfig{InsecureSkipVerify: flagInsecure}
		if flagCACert != "" {
			ca, err := os.ReadFile(flagCACert)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading CA cert: %v\n", err)
				os.Exit(1)
			}
			tlsCfg.CA = string(ca)
		}
		if flagCert != "" && flagKey != "" {
			cert, err := os.ReadFile(flagCert)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading cert: %v\n", err)
				os.Exit(1)
			}
			key, err := os.ReadFile(flagKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading key: %v\n", err)
				os.Exit(1)
			}
			tlsCfg.Cert = string(cert)
			tlsCfg.Key = string(key)
		}
		var err error
		client, err = client.TLSConfig(tlsCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "TLS config error: %v\n", err)
			os.Exit(1)
		}
	}

	if flagProxy != "" {
		client = client.Proxy(flagProxy)
	}

	if flagConnectTo != "" {
		client = client.ConnectTo(flagConnectTo)
	}

	if flagRetry > 0 {
		client = client.Retry(flagRetry, flagRetryWait, flagRetryFactor)
	}

	if flagNoFollow {
		client = client.RedirectPolicy(0)
	} else if flagMaxRedirects != 10 {
		client = client.RedirectPolicy(flagMaxRedirects)
	}

	if flagVerbose >= 3 {
		client = client.TraceToStdout(commonshttp.TraceAll)
	} else if flagVerbose >= 2 {
		client = client.TraceToStdout(commonshttp.TraceHeaders)
	}

	return client, collector
}

func resolveBody(items *parse.ParsedItems) (io.Reader, string, error) {
	if len(flagForm) > 0 {
		vals := url.Values{}
		for _, f := range flagForm {
			if k, v, ok := strings.Cut(f, "="); ok {
				vals.Set(k, v)
			}
		}
		return strings.NewReader(vals.Encode()), "application/x-www-form-urlencoded", nil
	}

	if flagData != "" {
		if strings.HasPrefix(flagData, "@") {
			data, err := os.ReadFile(flagData[1:])
			if err != nil {
				return nil, "", fmt.Errorf("reading file %s: %w", flagData[1:], err)
			}
			return bytes.NewReader(data), "", nil
		}
		return strings.NewReader(flagData), "", nil
	}

	if items.HasBody() {
		data, err := items.BodyJSON()
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(data), "application/json", nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return os.Stdin, "", nil
	}

	return nil, "", nil
}
