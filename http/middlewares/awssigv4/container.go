package awssigv4

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultContainerEndpoint = "http://169.254.170.2"
	envContainerRelativeURI  = "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"
	envContainerFullURI      = "AWS_CONTAINER_CREDENTIALS_FULL_URI"
	envContainerTokenFile    = "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE"
	envContainerToken        = "AWS_CONTAINER_AUTHORIZATION_TOKEN"
)

// ContainerOptions configures ContainerProvider. Empty fields fall back to the AWS_CONTAINER_* environment variables.
type ContainerOptions struct {
	// Endpoint is the base the relative URI is appended to; it defaults to http://169.254.170.2.
	Endpoint string
	// HTTPClient defaults to a client with a one second connect timeout and a five second overall timeout.
	HTTPClient *http.Client
}

// ContainerError is returned when the container credentials endpoint answers with a non-200 status.
type ContainerError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *ContainerError) Error() string {
	return fmt.Sprintf("container credentials endpoint returned HTTP %d (code %q): %s", e.StatusCode, e.Code, e.Message)
}

// ContainerProvider reads ECS task-role and EKS Pod Identity credentials from the container credentials endpoint
// named by AWS_CONTAINER_CREDENTIALS_RELATIVE_URI or AWS_CONTAINER_CREDENTIALS_FULL_URI. The authorization token
// file is re-read on every retrieval because projected tokens rotate.
type ContainerProvider struct{ options ContainerOptions }

func NewContainerProvider(options ContainerOptions) *ContainerProvider {
	return &ContainerProvider{options: options}
}

// containerConfigured reports whether the environment names a container credentials endpoint.
func containerConfigured() bool {
	return os.Getenv(envContainerRelativeURI) != "" || os.Getenv(envContainerFullURI) != ""
}

func (p *ContainerProvider) Retrieve(ctx context.Context) (Credentials, error) {
	target, err := p.targetURL()
	if err != nil {
		return Credentials{}, err
	}
	token, err := containerAuthorization()
	if err != nil {
		return Credentials{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Credentials{}, fmt.Errorf("build container credentials request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	client := p.options.HTTPClient
	if client == nil {
		client = defaultIMDSClient()
		defer client.CloseIdleConnections()
	}
	// Copy so the caller-owned client is not mutated; a redirect could downgrade to http and leak the token.
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := noRedirect.Do(request)
	if err != nil {
		return Credentials{}, fmt.Errorf("container credentials request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxSTSBodyBytes))
	if err != nil {
		return Credentials{}, fmt.Errorf("read container credentials response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Credentials{}, parseContainerError(response.StatusCode, body)
	}
	return parseContainerCredentials(body)
}

func (p *ContainerProvider) targetURL() (string, error) {
	if relative := os.Getenv(envContainerRelativeURI); relative != "" {
		if !strings.HasPrefix(relative, "/") || strings.HasPrefix(relative, "//") || strings.ContainsAny(relative, "@\\") {
			return "", fmt.Errorf("%s: must be an absolute path without scheme or authority, got %q", envContainerRelativeURI, relative)
		}
		return strings.TrimRight(firstNonEmpty(p.options.Endpoint, defaultContainerEndpoint), "/") + relative, nil
	}
	full := os.Getenv(envContainerFullURI)
	if full == "" {
		return "", fmt.Errorf("container credentials require %s or %s", envContainerRelativeURI, envContainerFullURI)
	}
	if err := validateContainerFullURI(full); err != nil {
		return "", fmt.Errorf("%s: %w", envContainerFullURI, err)
	}
	return full, nil
}

func validateContainerFullURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if containerHostAllowedOverHTTP(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("http is only allowed for loopback, 169.254.170.2, 169.254.170.23 and fd00:ec2::23 hosts, got %q", parsed.Hostname())
	default:
		return fmt.Errorf("unsupported scheme %q, expected https or http", parsed.Scheme)
	}
}

func containerHostAllowedOverHTTP(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, allowed := range []string{"169.254.170.2", "169.254.170.23", "fd00:ec2::23"} {
		if ip.Equal(net.ParseIP(allowed)) {
			return true
		}
	}
	return ip.IsLoopback()
}

// containerAuthorization returns the Authorization header value: the trimmed token file contents when
// AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE is set, otherwise AWS_CONTAINER_AUTHORIZATION_TOKEN (possibly empty).
func containerAuthorization() (string, error) {
	file := os.Getenv(envContainerTokenFile)
	if file == "" {
		return os.Getenv(envContainerToken), nil
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read container authorization token file: %w", err)
	}
	return strings.TrimSpace(string(contents)), nil
}

func parseContainerError(status int, body []byte) error {
	var parsed struct{ Code, Message string }
	if err := json.Unmarshal(body, &parsed); err != nil {
		return &ContainerError{StatusCode: status, Message: strings.TrimSpace(string(body))}
	}
	return &ContainerError{StatusCode: status, Code: parsed.Code, Message: parsed.Message}
}

func parseContainerCredentials(body []byte) (Credentials, error) {
	var parsed struct {
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string
		Token           string
		Expiration      string
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Credentials{}, fmt.Errorf("parse container credentials: %w", err)
	}
	expires, err := time.Parse(time.RFC3339, parsed.Expiration)
	if err != nil {
		return Credentials{}, fmt.Errorf("container credentials have invalid expiration %q: %w", parsed.Expiration, err)
	}
	credentials := Credentials{AccessKeyID: parsed.AccessKeyID, SecretAccessKey: parsed.SecretAccessKey, SessionToken: parsed.Token, Expires: expires}
	if err := credentials.validate(time.Now()); err != nil {
		return Credentials{}, fmt.Errorf("container endpoint returned unusable credentials: %w", err)
	}
	return credentials, nil
}
