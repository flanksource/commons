# bearer

Middleware that authenticates requests with `Authorization: Bearer <token>`, with the token supplied by a `TokenSource`.

`TokenSource` is the same one-method interface as `golang.org/x/oauth2.TokenSource` (`Token() (*oauth2.Token, error)`), so GCP credentials from `golang.org/x/oauth2/google`, the `azuread` package in this repository, or any other oauth2 source plug in directly.

Behaviour:

- Tokens are cached and refreshed shortly before expiry (`oauth2.ReuseTokenSource` semantics, safe for concurrent use). A source that returns a token without an expiry is cached forever.
- If the source returns an error, or an empty access token, the request fails and is never sent unauthenticated.
- If the request already has an `Authorization` header the request fails instead of silently overwriting it, so this middleware cannot be combined with `Client.Auth` or `Client.OAuth` on the same client.
- `Client.BearerToken` installs the middleware outside the HAR middlewares, so HAR captures the header (redacted by the HAR redaction policy) and trace redaction of `Authorization` keeps working.

## Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/flanksource/commons/http"
	"golang.org/x/oauth2"
)

type staticToken string

func (s staticToken) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(s)}, nil
}

func main() {
	client := http.NewClient().BearerToken(staticToken("my-token"))

	resp, err := client.R(context.Background()).Get("https://api.example.com/v1/things")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
}
```

With Google application default credentials:

```go
creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
if err != nil {
	return err
}
client := http.NewClient().BearerToken(creds.TokenSource)
```
