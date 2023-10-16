package middlewares

import (
	"fmt"
	netHttp "net/http"
	"time"

	"github.com/flanksource/commons/hash"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func NewOauthTransport(clientID, clientSecret, tokenURL string, scopes ...string) *oauthConfig {
	return &oauthConfig{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		scopes:       scopes,
		cache:        cache.New(time.Minute*15, time.Hour),
	}
}

type oauthConfig struct {
	clientID     string
	clientSecret string
	tokenURL     string
	scopes       []string
	cache        *cache.Cache
}

func (t *oauthConfig) RoundTripper(rt netHttp.RoundTripper) netHttp.RoundTripper {
	return RoundTripperFunc(func(ogRequest *netHttp.Request) (*netHttp.Response, error) {
		config := clientcredentials.Config{
			ClientID:     t.clientID,
			ClientSecret: t.clientSecret,
			TokenURL:     t.tokenURL,
			Scopes:       t.scopes,
		}

		cacheKey := oauthCacheKey(t.clientID, t.clientSecret, t.tokenURL, t.scopes)
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
