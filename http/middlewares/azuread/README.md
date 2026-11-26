# azuread

SDK-free Azure AD (Microsoft Entra ID) credentials that produce access tokens for the `bearer` middleware. No Azure SDK is imported; tokens are fetched with plain HTTP.

Every credential implements `Credential` (`Token(ctx, scope)`) and takes a scope such as `https://vault.azure.net/.default` or `https://api.loganalytics.io/.default`. `TokenSource(cred, scope)` adapts a credential to a `bearer.TokenSource`; caching and early refresh are done by the bearer middleware.

| Credential | Source | Notes |
| --- | --- | --- |
| `ClientSecret` | `{authority}/{tenant}/oauth2/v2.0/token`, `client_credentials` grant | `Authority` defaults to `https://login.microsoftonline.com`; override for sovereign clouds. `ClientSecretFromEnv` reads `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_AUTHORITY_HOST`. |
| `WorkloadIdentity` | same endpoint, `client_assertion` of type `jwt-bearer` | The federated token file is re-read on every token request because it is rotated. `WorkloadIdentityFromEnv` reads `AZURE_FEDERATED_TOKEN_FILE`, `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_AUTHORITY_HOST`. |
| `ManagedIdentity` | IMDS `http://169.254.169.254/metadata/identity/oauth2/token` with `Metadata: true`, or `IDENTITY_ENDPOINT` + `IDENTITY_HEADER` (App Service, Container Apps) when set | `ClientID` selects a user-assigned identity. Uses a 2 second timeout so an absent endpoint fails quickly. |
| `DefaultChain()` | environment client secret, then workload identity, then managed identity | When every credential fails the error joins one error per credential, each prefixed with its name. |

Failures never produce an empty token: HTTP errors include the AAD `error` and `error_description`, and a response without `access_token` or an expiry is rejected. `expires_in` and `expires_on` are accepted as numbers or quoted numbers.

Azure AD authorities require HTTPS, with HTTP allowed for loopback endpoints. Token requests reject redirects, including managed identity requests.

## Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/flanksource/commons/http"
	"github.com/flanksource/commons/http/middlewares/azuread"
)

func main() {
	const scope = "https://vault.azure.net/.default"

	// Explicit client secret.
	cred := azuread.ClientSecret{TenantID: "tenant-id", ClientID: "client-id", ClientSecret: "secret"}
	client := http.NewClient().BearerToken(azuread.TokenSource(cred, scope))

	// Or discover credentials from the environment / workload identity / managed identity.
	client = http.NewClient().BearerToken(azuread.DefaultTokenSource(scope))

	resp, err := client.R(context.Background()).Get("https://myvault.vault.azure.net/keys?api-version=7.4")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
}
```
