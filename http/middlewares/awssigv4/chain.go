package awssigv4

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type chainProvider []CredentialsProvider

// Chain returns a provider that yields the first successful provider's credentials. When every provider fails
// the returned error joins each failure, labelled with the provider's position and type.
func Chain(providers ...CredentialsProvider) CredentialsProvider {
	return chainProvider(providers)
}

func (c chainProvider) Retrieve(ctx context.Context) (Credentials, error) {
	if len(c) == 0 {
		return Credentials{}, fmt.Errorf("credentials chain has no providers")
	}
	var failures []error
	for i, provider := range c {
		if provider == nil {
			failures = append(failures, fmt.Errorf("chain[%d]: nil provider", i))
			continue
		}
		credentials, err := provider.Retrieve(ctx)
		if err == nil {
			return credentials, nil
		}
		failures = append(failures, fmt.Errorf("chain[%d] %T: %w", i, provider, err))
	}
	return Credentials{}, fmt.Errorf("no credentials provider succeeded: %w", errors.Join(failures...))
}

// DefaultChainOptions tunes DefaultChain.
type DefaultChainOptions struct {
	HTTPClient   *http.Client
	STSEndpoint  string
	IMDSEndpoint string
}

type DefaultChainOption func(*DefaultChainOptions)

func WithHTTPClient(client *http.Client) DefaultChainOption {
	return func(o *DefaultChainOptions) { o.HTTPClient = client }
}

func WithSTSEndpoint(endpoint string) DefaultChainOption {
	return func(o *DefaultChainOptions) { o.STSEndpoint = endpoint }
}

func WithIMDSEndpoint(endpoint string) DefaultChainOption {
	return func(o *DefaultChainOptions) { o.IMDSEndpoint = endpoint }
}

// DefaultChain resolves credentials from the environment, then web identity (only when AWS_WEB_IDENTITY_TOKEN_FILE
// and AWS_ROLE_ARN are set), then container credentials (only when AWS_CONTAINER_CREDENTIALS_RELATIVE_URI or
// AWS_CONTAINER_CREDENTIALS_FULL_URI is set), then IMDS. The result is expiry-aware.
func DefaultChain(region string, opts ...DefaultChainOption) *CredentialsCache {
	var o DefaultChainOptions
	for _, opt := range opts {
		opt(&o)
	}
	providers := []CredentialsProvider{EnvironmentProvider{}}
	if webIdentityConfigured() {
		providers = append(providers, NewWebIdentityProvider(WebIdentityOptions{Region: region, Endpoint: o.STSEndpoint, HTTPClient: o.HTTPClient}))
	}
	if containerConfigured() {
		providers = append(providers, NewContainerProvider(ContainerOptions{HTTPClient: o.HTTPClient}))
	}
	providers = append(providers, NewIMDSProvider(IMDSOptions{Endpoint: o.IMDSEndpoint, HTTPClient: o.HTTPClient}))
	return NewCredentialsCache(Chain(providers...))
}
