package awssigv4

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// AssumeRoleOptions configures AssumeRoleProvider. Region, Base, RoleARN and SessionName are required.
type AssumeRoleOptions struct {
	Region string
	// Endpoint overrides https://sts.<region>.amazonaws.com/ (for example a LocalStack URL).
	Endpoint string
	// HTTPClient defaults to http.DefaultClient.
	HTTPClient *http.Client
	// Base signs the AssumeRole request.
	Base        CredentialsProvider
	RoleARN     string
	SessionName string
	ExternalID  string
	// Duration is sent as DurationSeconds when positive; otherwise STS applies its own default.
	Duration time.Duration
}

// AssumeRoleProvider obtains temporary credentials from STS AssumeRole using the Query API.
type AssumeRoleProvider struct{ options AssumeRoleOptions }

func NewAssumeRoleProvider(options AssumeRoleOptions) *AssumeRoleProvider {
	return &AssumeRoleProvider{options: options}
}

func (p *AssumeRoleProvider) Retrieve(ctx context.Context) (Credentials, error) {
	o := p.options
	switch {
	case o.Base == nil:
		return Credentials{}, fmt.Errorf("AssumeRoleProvider requires a base credentials provider")
	case o.RoleARN == "":
		return Credentials{}, fmt.Errorf("AssumeRoleProvider requires a role ARN")
	case o.SessionName == "":
		return Credentials{}, fmt.Errorf("AssumeRoleProvider requires a session name")
	}
	params := url.Values{"RoleArn": {o.RoleARN}, "RoleSessionName": {o.SessionName}}
	if o.ExternalID != "" {
		params.Set("ExternalId", o.ExternalID)
	}
	durationSeconds(params, o.Duration)
	return stsCall{Action: "AssumeRole", Region: o.Region, Endpoint: o.Endpoint, Client: o.HTTPClient, Signer: o.Base, Params: params}.do(ctx)
}
