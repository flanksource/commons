package middlewares

import (
	"fmt"
	netHttp "net/http"
	"net/url"
	"time"

	"github.com/flanksource/commons/hash"
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
}

func (c *OauthConfig) AuthStyleInHeader() *OauthConfig {
	c.AuthStyle = AuthStyleInHeader
	return c
}

func (c *OauthConfig) AuthStyleInParams() *OauthConfig {
	c.AuthStyle = AuthStyleInParams
	return c
}

func (c *OauthConfig) getSanitizedSecret() string {
	if len(c.ClientSecret) <= 4 {
		return c.ClientSecret
	}
	return c.ClientSecret[0:4] + "****"
}

func (c OauthConfig) String() string {
	return fmt.Sprintf("url=%s id=%s, secret=%s scopes=%s params=%s", c.TokenURL, c.ClientID, c.getSanitizedSecret(), c.Scopes, c.Params)
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
		}

		var err error
		if token == nil {
			token, err = config.Token(ogRequest.Context())
			if err != nil {
				return nil, fmt.Errorf("error fetching oauth access token: %w", err)
			}

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
