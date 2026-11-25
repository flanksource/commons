package awssigv4

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// WebIdentityOptions configures WebIdentityProvider. Empty TokenFile, RoleARN and SessionName fall back to
// AWS_WEB_IDENTITY_TOKEN_FILE, AWS_ROLE_ARN and AWS_ROLE_SESSION_NAME. Region is required.
type WebIdentityOptions struct {
	Region string
	// Endpoint overrides https://sts.<region>.amazonaws.com/.
	Endpoint   string
	HTTPClient *http.Client
	TokenFile  string
	RoleARN    string
	// SessionName defaults to AWS_ROLE_SESSION_NAME, then to "commons-awssigv4-<unix nanoseconds>".
	SessionName string
	Duration    time.Duration
}

// WebIdentityProvider obtains credentials with the unsigned STS AssumeRoleWithWebIdentity call.
// The token file is re-read on every retrieval because projected tokens rotate.
type WebIdentityProvider struct{ options WebIdentityOptions }

func NewWebIdentityProvider(options WebIdentityOptions) *WebIdentityProvider {
	return &WebIdentityProvider{options: options}
}

// webIdentityConfigured reports whether the environment names both a token file and a role.
func webIdentityConfigured() bool {
	return os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" && os.Getenv("AWS_ROLE_ARN") != ""
}

func (p *WebIdentityProvider) Retrieve(ctx context.Context) (Credentials, error) {
	o := p.options
	tokenFile := firstNonEmpty(o.TokenFile, os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"))
	roleARN := firstNonEmpty(o.RoleARN, os.Getenv("AWS_ROLE_ARN"))
	if tokenFile == "" || roleARN == "" {
		return Credentials{}, fmt.Errorf("web identity requires a token file (AWS_WEB_IDENTITY_TOKEN_FILE) and role ARN (AWS_ROLE_ARN)")
	}
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return Credentials{}, fmt.Errorf("read web identity token file: %w", err)
	}
	if strings.TrimSpace(string(token)) == "" {
		return Credentials{}, fmt.Errorf("web identity token file %s is empty", tokenFile)
	}
	sessionName := firstNonEmpty(o.SessionName, os.Getenv("AWS_ROLE_SESSION_NAME"), fmt.Sprintf("commons-awssigv4-%d", time.Now().UnixNano()))
	params := url.Values{"RoleArn": {roleARN}, "RoleSessionName": {sessionName}, "WebIdentityToken": {strings.TrimSpace(string(token))}}
	durationSeconds(params, o.Duration)
	return stsCall{Action: "AssumeRoleWithWebIdentity", Region: o.Region, Endpoint: o.Endpoint, Client: o.HTTPClient, Params: params}.do(ctx)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
