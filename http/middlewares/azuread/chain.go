package azuread

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/flanksource/commons/http/middlewares/bearer"
	"golang.org/x/oauth2"
)

// Chain tries each credential in order and returns the first token. When every
// credential fails, the error joins one error per credential, each prefixed by
// its name.
type Chain []NamedCredential

// NamedCredential labels a credential for error reporting.
type NamedCredential struct {
	Name string
	Credential
}

func (c Chain) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	if len(c) == 0 {
		return nil, errors.New("azuread: empty credential chain")
	}
	var errs []error
	for _, cred := range c {
		token, err := cred.Token(ctx, scope)
		if err == nil {
			return token, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", cred.Name, err))
	}
	return nil, fmt.Errorf("azuread: all credentials failed: %w", errors.Join(errs...))
}

// envCredential defers reading the environment until a token is requested, so a
// chain can be built before the environment is final and unset variables are
// reported as that credential's error.
type envCredential[C Credential] struct {
	load func() (C, error)
}

func (e envCredential[C]) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	cred, err := e.load()
	if err != nil {
		return nil, err
	}
	return cred.Token(ctx, scope)
}

// Env is the client secret credential configured from AZURE_TENANT_ID,
// AZURE_CLIENT_ID and AZURE_CLIENT_SECRET.
func Env() Credential {
	return envCredential[ClientSecret]{load: ClientSecretFromEnv}
}

// WorkloadIdentityEnv is the workload identity credential configured from
// AZURE_TENANT_ID, AZURE_CLIENT_ID and AZURE_FEDERATED_TOKEN_FILE.
func WorkloadIdentityEnv() Credential {
	return envCredential[WorkloadIdentity]{load: WorkloadIdentityFromEnv}
}

// DefaultChain tries environment client secret, then workload identity, then
// managed identity (using AZURE_CLIENT_ID, when set, as the user-assigned
// identity).
func DefaultChain() Chain {
	return Chain{
		{Name: "environment client secret", Credential: Env()},
		{Name: "workload identity", Credential: WorkloadIdentityEnv()},
		{Name: "managed identity", Credential: envManaged{}},
	}
}

type envManaged struct{}

func (envManaged) Token(ctx context.Context, scope string) (*oauth2.Token, error) {
	return ManagedIdentity{ClientID: os.Getenv("AZURE_CLIENT_ID")}.Token(ctx, scope)
}

// DefaultTokenSource returns a bearer.TokenSource for scope backed by DefaultChain.
func DefaultTokenSource(scope string) bearer.TokenSource {
	return TokenSource(DefaultChain(), scope)
}
