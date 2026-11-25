package azuread

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	// DefaultIMDSEndpoint is the Azure VM instance metadata token endpoint.
	DefaultIMDSEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"

	imdsAPIVersion       = "2018-02-01"
	appServiceAPIVersion = "2019-08-01"
	managedTimeout       = 2 * time.Second
)

// ManagedIdentity fetches tokens from the Azure managed identity endpoint. When
// IDENTITY_ENDPOINT and IDENTITY_HEADER are set (App Service, Container Apps,
// Functions) that endpoint is used, otherwise the VM IMDS endpoint.
type ManagedIdentity struct {
	// ClientID selects a user-assigned identity; empty uses the system-assigned one.
	ClientID string
	// Endpoint overrides the token endpoint (tests); it disables the
	// IDENTITY_ENDPOINT detection and uses the IMDS request format.
	Endpoint string
	// HTTPClient defaults to a client with a 2s timeout, since IMDS is
	// link-local and an absent endpoint must fail quickly.
	HTTPClient *http.Client
}

// defaultManagedClient bypasses environment proxies: Azure requires IMDS requests to skip them.
var defaultManagedClient = &http.Client{
	Timeout:   managedTimeout,
	Transport: &http.Transport{Proxy: nil, IdleConnTimeout: 30 * time.Second},
}

func (m ManagedIdentity) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	resource := strings.TrimSuffix(scope, "/.default")
	if resource == "" {
		return nil, fmt.Errorf("managed identity requires a scope, got %q", scope)
	}
	req, err := m.request(ctx, resource)
	if err != nil {
		return nil, err
	}
	client := m.HTTPClient
	if client == nil {
		client = defaultManagedClient
	}
	return doTokenRequest(client, req)
}

func (m ManagedIdentity) request(ctx context.Context, resource string) (*http.Request, error) {
	endpoint, header := os.Getenv("IDENTITY_ENDPOINT"), os.Getenv("IDENTITY_HEADER")
	appService := m.Endpoint == "" && endpoint != "" && header != ""
	query := url.Values{"resource": {resource}}
	switch {
	case appService:
		query.Set("api-version", appServiceAPIVersion)
	case m.Endpoint != "":
		endpoint, query = m.Endpoint, withVersion(query, imdsAPIVersion)
	default:
		endpoint, query = DefaultIMDSEndpoint, withVersion(query, imdsAPIVersion)
	}
	if m.ClientID != "" {
		query.Set("client_id", m.ClientID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if appService {
		req.Header.Set("X-IDENTITY-HEADER", header)
	} else {
		req.Header.Set("Metadata", "true")
	}
	return req, nil
}

func withVersion(q url.Values, version string) url.Values {
	q.Set("api-version", version)
	return q
}
