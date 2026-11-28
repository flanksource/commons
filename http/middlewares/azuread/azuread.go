// Package azuread provides SDK-free Azure AD (Microsoft Entra ID) credentials
// that produce access tokens for the bearer middleware.
package azuread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flanksource/commons/http/internal/httppolicy"
	"github.com/flanksource/commons/http/middlewares/bearer"
	"golang.org/x/oauth2"
)

// DefaultAuthority is the public-cloud Azure AD authority.
const DefaultAuthority = "https://login.microsoftonline.com"

const (
	requestTimeout   = 10 * time.Second
	maxResponseBytes = 1 << 20
)

// Credential fetches an access token for a scope such as
// "https://vault.azure.net/.default".
type Credential interface {
	Token(ctx context.Context, scope string) (*oauth2.Token, error)
}

// TokenSource adapts a Credential and scope to a bearer.TokenSource. Caching
// and early refresh are done by the bearer middleware, so the returned source
// performs a network call on every Token() invocation.
func TokenSource(cred Credential, scope string) bearer.TokenSource {
	return tokenSource{cred: cred, scope: scope}
}

type tokenSource struct {
	cred  Credential
	scope string
}

func (s tokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*requestTimeout)
	defer cancel()
	return s.cred.Token(ctx, s.scope)
}

func httpClientOrDefault(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: requestTimeout}
}

// tokenResponse covers both the AAD v2 endpoint and the managed identity
// endpoints, which differ in whether expires_in/expires_on are numbers or
// strings.
type tokenResponse struct {
	AccessToken      string   `json:"access_token"`
	TokenType        string   `json:"token_type"`
	ExpiresIn        flexInt  `json:"expires_in"`
	ExpiresOn        flexInt  `json:"expires_on"`
	Error            flexText `json:"error"`
	ErrorDescription string   `json:"error_description"`
	Message          string   `json:"message"`
}

// flexInt decodes a JSON number or a quoted number.
type flexInt int64

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid numeric value %q", s)
	}
	*f = flexInt(n)
	return nil
}

// flexText decodes a JSON string, or the raw JSON of any other value (managed
// identity endpoints return "error" as an object).
type flexText string

func (f *flexText) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*f = flexText(s)
		return nil
	}
	*f = flexText(string(b))
	return nil
}

// doTokenRequest sends req and converts the response into a token, surfacing
// the AAD error and error_description for non-2xx responses.
func doTokenRequest(client *http.Client, req *http.Request) (*oauth2.Token, error) {
	noRedirect := httppolicy.WithoutRedirects(httpClientOrDefault(client))
	resp, err := noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request to %s failed: %w", req.URL.Host, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading token response from %s: %w", req.URL.Host, err)
	}
	var parsed tokenResponse
	jsonErr := json.Unmarshal(body, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, tokenError(req, resp.StatusCode, parsed, body)
	}
	if jsonErr != nil {
		return nil, fmt.Errorf("parsing token response from %s: %w", req.URL.Host, jsonErr)
	}
	return parsed.toToken()
}

func tokenError(req *http.Request, status int, parsed tokenResponse, body []byte) error {
	detail := strings.TrimSpace(string(parsed.Error))
	if parsed.ErrorDescription != "" {
		detail += ": " + strings.TrimSpace(parsed.ErrorDescription)
	}
	if detail == "" {
		detail = strings.TrimSpace(parsed.Message)
	}
	if detail == "" {
		detail = strings.TrimSpace(string(body))
		if len(detail) > 512 {
			detail = detail[:512] + "..."
		}
	}
	return fmt.Errorf("token endpoint %s returned HTTP %d: %s", req.URL.Host, status, detail)
}

func (r tokenResponse) toToken() (*oauth2.Token, error) {
	if r.AccessToken == "" {
		return nil, errors.New("token response has no access_token")
	}
	var expiry time.Time
	switch {
	case r.ExpiresOn > 0:
		expiry = time.Unix(int64(r.ExpiresOn), 0)
	case r.ExpiresIn > 0:
		expiry = time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)
	default:
		return nil, errors.New("token response has neither expires_in nor expires_on")
	}
	return &oauth2.Token{AccessToken: r.AccessToken, TokenType: r.TokenType, Expiry: expiry}, nil
}
