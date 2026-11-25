// Package bearer provides a middleware that authenticates requests with an
// "Authorization: Bearer <token>" header obtained from a TokenSource.
package bearer

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/flanksource/commons/http/middlewares"
	"golang.org/x/oauth2"
)

// TokenSource is compatible with golang.org/x/oauth2.TokenSource, so
// credentials from golang.org/x/oauth2/google (or any oauth2 source) can be
// passed in directly.
type TokenSource interface {
	Token() (*oauth2.Token, error)
}

// New returns a middleware that sets "Authorization: Bearer <token>" on every
// request. Tokens are cached and refreshed shortly before expiry
// (oauth2.ReuseTokenSource semantics, safe for concurrent use).
//
// The middleware fails fast: if the token source errors or returns an empty
// token the request is not sent, and if the request already carries an
// Authorization header it is rejected rather than silently overwritten.
func New(src TokenSource) middlewares.Middleware {
	var cached oauth2.TokenSource
	if src != nil {
		cached = oauth2.ReuseTokenSource(nil, src)
	}
	return func(next http.RoundTripper) http.RoundTripper {
		return middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			header, err := authorization(cached, req)
			if err != nil {
				if req.Body != nil {
					_ = req.Body.Close()
				}
				return nil, err
			}
			clone := req.Clone(req.Context())
			clone.Header.Set("Authorization", header)
			return next.RoundTrip(clone)
		})
	}
}

func authorization(src oauth2.TokenSource, req *http.Request) (string, error) {
	if src == nil {
		return "", errors.New("bearer: nil TokenSource")
	}
	if existing := req.Header.Get("Authorization"); existing != "" {
		return "", errors.New("bearer: request already has an Authorization header; refusing to overwrite it")
	}
	token, err := src.Token()
	if err != nil {
		return "", fmt.Errorf("bearer: fetching token: %w", err)
	}
	if token == nil || token.AccessToken == "" {
		return "", errors.New("bearer: token source returned an empty access token")
	}
	return "Bearer " + token.AccessToken, nil
}
