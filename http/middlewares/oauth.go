package middlewares

import (
	"context"
	"fmt"
	netHttp "net/http"
	"net/url"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/hash"
	"github.com/flanksource/commons/logger"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func NewOauthTransport(config OauthConfig) *oauthRoundTripper {
	return &oauthRoundTripper{OauthConfig: config, cache: cache.New(time.Minute*15, time.Hour)}
}

type AuthStyle oauth2.AuthStyle

var AuthStyleInHeader = AuthStyle(oauth2.AuthStyleInHeader)
var AuthStyleInParams = AuthStyle(oauth2.AuthStyleInParams)
var AuthStyleAutoDetect = AuthStyle(oauth2.AuthStyleAutoDetect)

type OauthConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
	Params       map[string]string
	AuthStyle    AuthStyle
	Tracer       func(msg string)
	// TokenTransport wraps the HTTP transport used for OAuth token requests.
	// When set, the token fetch HTTP call is routed through this middleware,
	// allowing HAR capture of the token request without a circular import.
	TokenTransport Middleware
}

func (c OauthConfig) Pretty() api.Text {
	t := clicky.Text(c.TokenURL)
	t = t.Space().
		Append("id=", "text-muted").Append(c.ClientID).
		Append(" scopes=", "text-muted").Append(c.Scopes).
		Append(c.Params)

	return t
}

func (c OauthConfig) String() string {
	return c.Pretty().String()
}

func (c *OauthConfig) AuthStyleInHeader() *OauthConfig {
	c.AuthStyle = AuthStyleInHeader
	return c
}

func (c *OauthConfig) AuthStyleInParams() *OauthConfig {
	c.AuthStyle = AuthStyleInParams
	return c
}

type oauthRoundTripper struct {
	OauthConfig
	cache *cache.Cache
}

func toUrlValues(m map[string]string) url.Values {
	values := url.Values{}
	for k, v := range m {
		values[k] = []string{v}
	}
	return values
}

func (t *oauthRoundTripper) trace(format string, args ...any) {
	logger.V(logger.Trace4).Infof(format, args...)
	if t.Tracer != nil {
		t.Tracer(fmt.Sprintf(format, args...))
	}
}

func (t *oauthRoundTripper) RoundTripper(rt netHttp.RoundTripper) netHttp.RoundTripper {
	return RoundTripperFunc(func(ogRequest *netHttp.Request) (*netHttp.Response, error) {
		config := clientcredentials.Config{
			ClientID:       t.ClientID,
			ClientSecret:   t.ClientSecret,
			TokenURL:       t.TokenURL,
			Scopes:         t.Scopes,
			EndpointParams: toUrlValues(t.Params),
			AuthStyle:      oauth2.AuthStyle(t.AuthStyle),
		}

		cacheKey := oauthCacheKey(t.ClientID, t.ClientSecret, t.TokenURL, t.Scopes)
		var token *oauth2.Token
		if val, ok := t.cache.Get(cacheKey); ok {
			token, _ = val.(*oauth2.Token)
			t.trace("oauth: using cached token (expires %s)", token.Expiry.Format(time.RFC3339))
		}

		var err error
		if token == nil {
			t.trace("fetching oauth token from %s", t.Pretty().ANSI())
			ctx := ogRequest.Context()
			if t.TokenTransport != nil {
				ctx = context.WithValue(ctx, oauth2.HTTPClient, &netHttp.Client{
					Transport: t.TokenTransport(netHttp.DefaultTransport),
				})
			}
			token, err = config.Token(ctx)
			if err != nil {
				return nil, fmt.Errorf("error fetching oauth access token: %w", err)
			}
			if !token.Valid() {
				return nil, fmt.Errorf("fetched invalid oauth token: type=%s expires in=%s", token.TokenType, time.Until(token.Expiry))
			}
			t.trace("oauth: token acquired (expires %s): access=%s, refresh=%s", token.Expiry.Format(time.RFC3339), logger.PrintableSecret(token.AccessToken), logger.PrintableSecret(token.RefreshToken))
			t.cache.Set(cacheKey, token, time.Until(token.Expiry))
		}

		request := ogRequest.Clone(ogRequest.Context())
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

		return rt.RoundTrip(request)
	})
}

func oauthCacheKey(ClientID, clientSecret, tokenURL string, scopes []string) string {
	return hash.Sha256Hex(fmt.Sprintf("%s:%s:%s:%s", ClientID, clientSecret, tokenURL, scopes))
}
