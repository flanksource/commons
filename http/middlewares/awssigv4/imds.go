package awssigv4

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultIMDSEndpoint = "http://169.254.169.254"
	imdsTokenTTL        = "21600"
	imdsRolesPath       = "/latest/meta-data/iam/security-credentials/"
)

// IMDSOptions configures IMDSProvider.
type IMDSOptions struct {
	// Endpoint defaults to AWS_EC2_METADATA_SERVICE_ENDPOINT, then http://169.254.169.254.
	Endpoint string
	// HTTPClient defaults to a client with a one second connect timeout and a five second overall timeout.
	HTTPClient *http.Client
}

// IMDSProvider reads instance-role credentials from the EC2 instance metadata service using IMDSv2 only.
type IMDSProvider struct{ options IMDSOptions }

func NewIMDSProvider(options IMDSOptions) *IMDSProvider { return &IMDSProvider{options: options} }

func defaultIMDSClient() *http.Client {
	dialer := &net.Dialer{Timeout: time.Second}
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DialContext: dialer.DialContext}}
}

func (p *IMDSProvider) Retrieve(ctx context.Context) (Credentials, error) {
	if strings.EqualFold(os.Getenv("AWS_EC2_METADATA_DISABLED"), "true") {
		return Credentials{}, fmt.Errorf("IMDS disabled by AWS_EC2_METADATA_DISABLED")
	}
	endpoint := strings.TrimRight(firstNonEmpty(p.options.Endpoint, os.Getenv("AWS_EC2_METADATA_SERVICE_ENDPOINT"), defaultIMDSEndpoint), "/")
	client := p.options.HTTPClient
	if client == nil {
		client = defaultIMDSClient()
		defer client.CloseIdleConnections()
	}
	token, err := imdsRequest(ctx, client, http.MethodPut, endpoint+"/latest/api/token", map[string]string{"X-aws-ec2-metadata-token-ttl-seconds": imdsTokenTTL})
	if err != nil {
		return Credentials{}, fmt.Errorf("IMDS token: %w", err)
	}
	headers := map[string]string{"X-aws-ec2-metadata-token": token}
	roles, err := imdsRequest(ctx, client, http.MethodGet, endpoint+imdsRolesPath, headers)
	if err != nil {
		return Credentials{}, fmt.Errorf("IMDS role name: %w", err)
	}
	role, _, _ := strings.Cut(strings.TrimSpace(roles), "\n")
	if role == "" {
		return Credentials{}, fmt.Errorf("IMDS reports no instance role")
	}
	document, err := imdsRequest(ctx, client, http.MethodGet, endpoint+imdsRolesPath+strings.TrimSpace(role), headers)
	if err != nil {
		return Credentials{}, fmt.Errorf("IMDS credentials for role %q: %w", role, err)
	}
	return parseIMDSCredentials(document)
}

func imdsRequest(ctx context.Context, client *http.Client, method, target string, headers map[string]string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return "", err
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxSTSBodyBytes))
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s %s returned HTTP %d", method, target, response.StatusCode)
	}
	return string(body), nil
}

func parseIMDSCredentials(document string) (Credentials, error) {
	var parsed struct {
		Code            string
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string
		Token           string
		Expiration      string
	}
	if err := json.Unmarshal([]byte(document), &parsed); err != nil {
		return Credentials{}, fmt.Errorf("parse IMDS credentials: %w", err)
	}
	if parsed.Code != "Success" {
		return Credentials{}, fmt.Errorf("IMDS credentials code %q is not Success", parsed.Code)
	}
	expires, err := time.Parse(time.RFC3339, parsed.Expiration)
	if err != nil {
		return Credentials{}, fmt.Errorf("IMDS credentials have invalid expiration %q: %w", parsed.Expiration, err)
	}
	credentials := Credentials{AccessKeyID: parsed.AccessKeyID, SecretAccessKey: parsed.SecretAccessKey, SessionToken: parsed.Token, Expires: expires}
	if err := credentials.validate(time.Now()); err != nil {
		return Credentials{}, fmt.Errorf("IMDS returned unusable credentials: %w", err)
	}
	return credentials, nil
}
