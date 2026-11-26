package azuread

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/flanksource/commons/http/internal/httppolicy"
	"golang.org/x/oauth2"
)

const jwtBearerAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// ClientSecret authenticates an app registration with a client secret using
// the OAuth2 client-credentials grant.
type ClientSecret struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	// Authority defaults to DefaultAuthority; override for sovereign clouds.
	Authority string
	// HTTPClient defaults to a client with a 10s timeout.
	HTTPClient *http.Client
}

// ClientSecretFromEnv reads AZURE_TENANT_ID, AZURE_CLIENT_ID,
// AZURE_CLIENT_SECRET and the optional AZURE_AUTHORITY_HOST.
func ClientSecretFromEnv() (ClientSecret, error) {
	vars, err := requireEnv("AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET")
	if err != nil {
		return ClientSecret{}, err
	}
	return ClientSecret{
		TenantID: vars[0], ClientID: vars[1], ClientSecret: vars[2],
		Authority: os.Getenv("AZURE_AUTHORITY_HOST"),
	}, nil
}

func (c ClientSecret) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	if c.TenantID == "" || c.ClientID == "" || c.ClientSecret == "" {
		return nil, errors.New("client secret credential requires TenantID, ClientID and ClientSecret")
	}
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"scope":         {scope},
	}
	return postForm(ctx, c.HTTPClient, c.Authority, c.TenantID, form)
}

// WorkloadIdentity authenticates with a federated (projected service account)
// token, as used by AKS workload identity.
type WorkloadIdentity struct {
	TenantID string
	ClientID string
	// TokenFile is the path of the federated token; it is re-read on every
	// token request because the file is rotated by the platform.
	TokenFile string
	Authority string
	// HTTPClient defaults to a client with a 10s timeout.
	HTTPClient *http.Client
}

// WorkloadIdentityFromEnv reads AZURE_TENANT_ID, AZURE_CLIENT_ID,
// AZURE_FEDERATED_TOKEN_FILE and the optional AZURE_AUTHORITY_HOST.
func WorkloadIdentityFromEnv() (WorkloadIdentity, error) {
	vars, err := requireEnv("AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_FEDERATED_TOKEN_FILE")
	if err != nil {
		return WorkloadIdentity{}, err
	}
	return WorkloadIdentity{
		TenantID: vars[0], ClientID: vars[1], TokenFile: vars[2],
		Authority: os.Getenv("AZURE_AUTHORITY_HOST"),
	}, nil
}

func (w WorkloadIdentity) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	if w.TenantID == "" || w.ClientID == "" || w.TokenFile == "" {
		return nil, errors.New("workload identity credential requires TenantID, ClientID and TokenFile")
	}
	raw, err := os.ReadFile(w.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading federated token file: %w", err)
	}
	assertion := strings.TrimSpace(string(raw))
	if assertion == "" {
		return nil, fmt.Errorf("federated token file %s is empty", w.TokenFile)
	}
	form := url.Values{
		"grant_type":            {"client_credentials"},
		"client_id":             {w.ClientID},
		"client_assertion_type": {jwtBearerAssertionType},
		"client_assertion":      {assertion},
		"scope":                 {scope},
	}
	return postForm(ctx, w.HTTPClient, w.Authority, w.TenantID, form)
}

func postForm(ctx context.Context, client *http.Client, authority, tenant string, form url.Values) (*oauth2.Token, error) {
	if authority == "" {
		authority = DefaultAuthority
	}
	base, err := url.Parse(authority)
	if err != nil {
		return nil, errors.New("azuread: invalid authority URL")
	}
	if base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return nil, errors.New("azuread: authority must have a host and cannot contain user information, a query or a fragment")
	}
	if !httppolicy.AllowsCredentials(base) {
		return nil, errors.New("azuread: authority requires HTTPS except on loopback HTTP endpoints")
	}
	endpoint := strings.TrimRight(base.String(), "/") + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return doTokenRequest(client, req)
}

// requireEnv returns the values of names, or an error listing every unset one.
func requireEnv(names ...string) ([]string, error) {
	values := make([]string, len(names))
	var missing []string
	for i, name := range names {
		if values[i] = os.Getenv(name); values[i] == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("environment variables not set: %s", strings.Join(missing, ", "))
	}
	return values, nil
}
