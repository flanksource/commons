package azuread_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/flanksource/commons/http/middlewares/azuread"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAzureAD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure AD Suite")
}

const vaultScope = "https://vault.azure.net/.default"

// aadServer records form posts and answers with the configured status/body.
type aadServer struct {
	*httptest.Server
	mu     sync.Mutex
	paths  []string
	forms  []url.Values
	status int
	body   string
}

func newAADServer(status int, body string) *aadServer {
	s := &aadServer{status: status, body: body}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		s.mu.Lock()
		s.paths = append(s.paths, r.URL.Path)
		s.forms = append(s.forms, r.PostForm)
		status, body := s.status, s.body
		s.mu.Unlock()
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	DeferCleanup(s.Close)
	return s
}

var _ = Describe("ClientSecret", func() {
	It("posts a client_credentials grant to the tenant endpoint and parses expires_in", func() {
		srv := newAADServer(200, `{"access_token":"tok","token_type":"Bearer","expires_in":3599}`)

		token, err := azuread.ClientSecret{
			TenantID: "tenant-1", ClientID: "client-1", ClientSecret: "s3cret", Authority: srv.URL + "/",
		}.Token(context.Background(), vaultScope)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.AccessToken).To(Equal("tok"))
		Expect(token.Valid()).To(BeTrue())
		Expect(srv.paths).To(Equal([]string{"/tenant-1/oauth2/v2.0/token"}))
		Expect(srv.forms[0]).To(Equal(url.Values{
			"grant_type": {"client_credentials"}, "client_id": {"client-1"},
			"client_secret": {"s3cret"}, "scope": {vaultScope},
		}))
	})

	It("surfaces the AAD error and error_description", func() {
		srv := newAADServer(401, `{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret"}`)

		_, err := azuread.ClientSecret{TenantID: "t", ClientID: "c", ClientSecret: "bad", Authority: srv.URL}.
			Token(context.Background(), vaultScope)

		Expect(err).To(MatchError(And(
			ContainSubstring("HTTP 401"),
			ContainSubstring("invalid_client"),
			ContainSubstring("AADSTS7000215"),
		)))
	})

	It("rejects a success response without an access token or expiry", func() {
		srv := newAADServer(200, `{"token_type":"Bearer"}`)
		_, err := azuread.ClientSecret{TenantID: "t", ClientID: "c", ClientSecret: "s", Authority: srv.URL}.
			Token(context.Background(), vaultScope)
		Expect(err).To(MatchError(ContainSubstring("no access_token")))

		srv.mu.Lock()
		srv.body = `{"access_token":"tok"}`
		srv.mu.Unlock()
		_, err = azuread.ClientSecret{TenantID: "t", ClientID: "c", ClientSecret: "s", Authority: srv.URL}.
			Token(context.Background(), vaultScope)
		Expect(err).To(MatchError(ContainSubstring("neither expires_in nor expires_on")))
	})

	It("requires tenant, client id and secret", func() {
		_, err := azuread.ClientSecret{TenantID: "t"}.Token(context.Background(), vaultScope)
		Expect(err).To(MatchError(ContainSubstring("requires TenantID, ClientID and ClientSecret")))
	})
})

var _ = Describe("WorkloadIdentity", func() {
	It("sends the federated token file contents as the client assertion and re-reads it on each request", func() {
		srv := newAADServer(200, `{"access_token":"tok","expires_in":"3600"}`)
		file := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(file, []byte("jwt-one\n"), 0o600)).To(Succeed())
		cred := azuread.WorkloadIdentity{TenantID: "t", ClientID: "c", TokenFile: file, Authority: srv.URL}

		_, err := cred.Token(context.Background(), vaultScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(file, []byte("jwt-two"), 0o600)).To(Succeed())
		_, err = cred.Token(context.Background(), vaultScope)
		Expect(err).NotTo(HaveOccurred())

		Expect(srv.forms[0]).To(Equal(url.Values{
			"grant_type": {"client_credentials"}, "client_id": {"c"}, "scope": {vaultScope},
			"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
			"client_assertion":      {"jwt-one"},
		}))
		Expect(srv.forms[1].Get("client_assertion")).To(Equal("jwt-two"))
	})

	It("fails when the token file is missing", func() {
		_, err := azuread.WorkloadIdentity{
			TenantID: "t", ClientID: "c", TokenFile: filepath.Join(GinkgoT().TempDir(), "absent"),
		}.Token(context.Background(), vaultScope)
		Expect(err).To(MatchError(ContainSubstring("reading federated token file")))
	})
})

var _ = Describe("ManagedIdentity", func() {
	var (
		srv     *httptest.Server
		headers http.Header
		query   url.Values
	)

	BeforeEach(func() {
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers, query = r.Header.Clone(), r.URL.Query()
			if r.Header.Get("Metadata") != "true" && r.Header.Get("X-IDENTITY-HEADER") == "" {
				w.WriteHeader(400)
				_, _ = fmt.Fprint(w, `{"error":"invalid_request","error_description":"Required metadata header not specified"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"access_token":"imds-tok","expires_on":"4102444800","resource":"x"}`)
		}))
		DeferCleanup(srv.Close)
	})

	It("calls IMDS with the Metadata header, resource and client_id, and parses expires_on", func() {
		token, err := azuread.ManagedIdentity{ClientID: "uami", Endpoint: srv.URL}.Token(context.Background(), vaultScope)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.AccessToken).To(Equal("imds-tok"))
		Expect(token.Expiry.Year()).To(Equal(2100))
		Expect(headers.Get("Metadata")).To(Equal("true"))
		Expect(query).To(Equal(url.Values{
			"api-version": {"2018-02-01"}, "resource": {"https://vault.azure.net"}, "client_id": {"uami"},
		}))
	})

	It("uses IDENTITY_ENDPOINT and IDENTITY_HEADER when present", func() {
		GinkgoT().Setenv("IDENTITY_ENDPOINT", srv.URL)
		GinkgoT().Setenv("IDENTITY_HEADER", "hdr-secret")

		_, err := azuread.ManagedIdentity{}.Token(context.Background(), vaultScope)

		Expect(err).NotTo(HaveOccurred())
		Expect(headers.Get("X-IDENTITY-HEADER")).To(Equal("hdr-secret"))
		Expect(query.Get("api-version")).To(Equal("2019-08-01"))
	})

	It("surfaces the endpoint error body", func() {
		failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `{"error":"invalid_request","error_description":"Identity not found"}`)
		}))
		DeferCleanup(failing.Close)

		_, err := azuread.ManagedIdentity{Endpoint: failing.URL}.Token(context.Background(), vaultScope)

		Expect(err).To(MatchError(And(ContainSubstring("invalid_request"), ContainSubstring("Identity not found"))))
	})
})

var _ = Describe("DefaultChain", func() {
	clearAzureEnv := func() {
		for _, name := range []string{
			"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET",
			"AZURE_FEDERATED_TOKEN_FILE", "AZURE_AUTHORITY_HOST", "IDENTITY_ENDPOINT", "IDENTITY_HEADER",
		} {
			GinkgoT().Setenv(name, "")
		}
	}

	It("uses the environment client secret first", func() {
		clearAzureEnv()
		srv := newAADServer(200, `{"access_token":"env-tok","expires_in":3600}`)
		GinkgoT().Setenv("AZURE_TENANT_ID", "t")
		GinkgoT().Setenv("AZURE_CLIENT_ID", "c")
		GinkgoT().Setenv("AZURE_CLIENT_SECRET", "s")
		GinkgoT().Setenv("AZURE_AUTHORITY_HOST", srv.URL)

		token, err := azuread.DefaultChain().Token(context.Background(), vaultScope)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.AccessToken).To(Equal("env-tok"))
	})

	It("falls through to workload identity when no client secret is set", func() {
		clearAzureEnv()
		srv := newAADServer(200, `{"access_token":"wi-tok","expires_in":3600}`)
		file := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(file, []byte("jwt"), 0o600)).To(Succeed())
		GinkgoT().Setenv("AZURE_TENANT_ID", "t")
		GinkgoT().Setenv("AZURE_CLIENT_ID", "c")
		GinkgoT().Setenv("AZURE_FEDERATED_TOKEN_FILE", file)
		GinkgoT().Setenv("AZURE_AUTHORITY_HOST", srv.URL)

		token, err := azuread.DefaultChain().Token(context.Background(), vaultScope)

		Expect(err).NotTo(HaveOccurred())
		Expect(token.AccessToken).To(Equal("wi-tok"))
	})

	It("joins one named error per credential when all fail", func() {
		clearAzureEnv()
		unreachable := azuread.Chain{
			{Name: "environment client secret", Credential: azuread.Env()},
			{Name: "workload identity", Credential: azuread.WorkloadIdentityEnv()},
			{Name: "managed identity", Credential: azuread.ManagedIdentity{Endpoint: "http://127.0.0.1:1"}},
		}

		_, err := unreachable.Token(context.Background(), vaultScope)

		Expect(err).To(MatchError(And(
			ContainSubstring("environment client secret: environment variables not set: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET"),
			ContainSubstring("workload identity: environment variables not set"),
			ContainSubstring("managed identity: token request to 127.0.0.1:1 failed"),
		)))
	})
})

var _ = Describe("TokenSource", func() {
	It("returns a bearer-compatible source that refetches on every call", func() {
		srv := newAADServer(200, `{"access_token":"tok","expires_in":3600}`)
		src := azuread.TokenSource(azuread.ClientSecret{TenantID: "t", ClientID: "c", ClientSecret: "s", Authority: srv.URL}, vaultScope)

		_, err := src.Token()
		Expect(err).NotTo(HaveOccurred())
		_, err = src.Token()
		Expect(err).NotTo(HaveOccurred())

		Expect(srv.paths).To(HaveLen(2))
	})
})
