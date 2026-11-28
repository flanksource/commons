package azuread_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/http/middlewares/azuread"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Azure credential endpoint policy", func() {
	DescribeTable("validates the AAD authority before contacting the endpoint",
		func(authority, expectedHost string) {
			var hosts []string
			client := &http.Client{Transport: middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				_ = req.Body.Close()
				hosts = append(hosts, req.URL.Host)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"example-token","expires_in":3600}`))}, nil
			})}
			credential := azuread.ClientSecret{
				TenantID: "example-tenant", ClientID: "example-client", ClientSecret: "example-secret",
				Authority: authority, HTTPClient: client,
			}

			token, err := credential.Token(context.Background(), vaultScope)

			if expectedHost == "" {
				Expect(err).To(MatchError(ContainSubstring("azuread: authority")))
				Expect(hosts).To(BeEmpty())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(token.AccessToken).To(Equal("example-token"))
				Expect(hosts).To(Equal([]string{expectedHost}))
			}
		},
		Entry("default HTTPS authority", "", "login.microsoftonline.com"),
		Entry("sovereign HTTPS authority", "https://login.microsoftonline.us/", "login.microsoftonline.us"),
		Entry("loopback IPv6", "http://[::1]:8080/", "[::1]:8080"),
		Entry("remote HTTP", "http://example.com", ""),
		Entry("relative authority", "//example.com", ""),
		Entry("unsupported scheme", "ftp://localhost", ""),
		Entry("missing host", "https:///path", ""),
		Entry("authority with a query", "https://example.com?x=1", ""),
	)

	DescribeTable("rejects credential endpoint redirects without changing the supplied client",
		func(path string, credential func(*http.Client) azuread.Credential) {
			var paths []string
			redirectChecks := 0
			client := &http.Client{
				CheckRedirect: func(*http.Request, []*http.Request) error {
					redirectChecks++
					return nil
				},
				Transport: middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
					if req.Body != nil {
						_ = req.Body.Close()
					}
					paths = append(paths, req.URL.Path)
					if req.URL.Path != path {
						return nil, errors.New("unexpected request to mock token endpoint")
					}
					return &http.Response{
						StatusCode: http.StatusPermanentRedirect, Header: http.Header{"Location": {"/next"}}, Body: http.NoBody,
					}, nil
				}),
			}

			_, err := credential(client).Token(context.Background(), vaultScope)

			Expect(err).To(MatchError(ContainSubstring("HTTP 308")))
			Expect(paths).To(Equal([]string{path}))
			Expect(redirectChecks).To(BeZero())
			Expect(client.CheckRedirect(nil, nil)).To(Succeed())
			Expect(redirectChecks).To(Equal(1))
		},
		Entry("AAD form POST", "/example-tenant/oauth2/v2.0/token", func(client *http.Client) azuread.Credential {
			return azuread.ClientSecret{
				TenantID: "example-tenant", ClientID: "example-client", ClientSecret: "example-secret",
				Authority: "https://example.com", HTTPClient: client,
			}
		}),
		Entry("App Service managed identity", "/identity", func(client *http.Client) azuread.Credential {
			GinkgoT().Setenv("IDENTITY_ENDPOINT", "http://localhost/identity")
			GinkgoT().Setenv("IDENTITY_HEADER", "example-header")
			return azuread.ManagedIdentity{HTTPClient: client}
		}),
	)
})
