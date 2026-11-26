package http

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

type staticTokenSource string

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(s), Expiry: time.Now().Add(time.Hour)}, nil
}

type tokenSourceFunc func() (*oauth2.Token, error)

func (f tokenSourceFunc) Token() (*oauth2.Token, error) { return f() }

type closeRecordingBody struct {
	io.Reader
	closes int
}

func (b *closeRecordingBody) Close() error {
	b.closes++
	return nil
}

var _ = Describe("Client.BearerToken", func() {
	const secretToken = "super-secret-bearer-token-value"

	var (
		seen      chan string
		serverURL string
	)

	BeforeEach(func() {
		seen = make(chan string, 4)
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			seen <- r.Header.Get("Authorization")
		}))
		DeferCleanup(server.Close)
		serverURL = server.URL
	})

	It("sends the bearer token and redacts it in the HAR capture", func() {
		collector := har.NewCollector(har.DefaultConfig())

		response, err := NewClient().
			HARCollector(collector).
			BearerToken(staticTokenSource(secretToken)).
			R(context.Background()).
			Get(serverURL)

		Expect(err).ToNot(HaveOccurred())
		Expect(response.Body.Close()).To(Succeed())
		Expect(<-seen).To(Equal("Bearer " + secretToken))

		entries := collector.Entries()
		Expect(entries).To(HaveLen(1))
		var authorization string
		for _, h := range entries[0].Request.Headers {
			if h.Name == "Authorization" {
				authorization = h.Value
			}
		}
		Expect(authorization).ToNot(BeEmpty(), "HAR sees the header because bearer wraps the HAR middleware")
		Expect(authorization).ToNot(ContainSubstring(secretToken))
	})

	It("refuses a request that already carries an Authorization header", func() {
		_, err := NewClient().
			BearerToken(staticTokenSource(secretToken)).
			R(context.Background()).
			Header("Authorization", "Basic abc").
			Get(serverURL)

		Expect(err).To(MatchError(ContainSubstring("already has an Authorization header")))
		Expect(seen).To(BeEmpty())
	})
})

var _ = Describe("client authentication conflicts", func() {
	DescribeTable("rejects bearer with OAuth or SigV4 before either credential is fetched and closes the body",
		func(configure func(*Client, tokenSourceFunc, OauthConfig, awssigv4.Config), conflictingMethod string) {
			var credentialCalls int
			bearer := tokenSourceFunc(func() (*oauth2.Token, error) {
				credentialCalls++
				return nil, errors.New("unexpected bearer token request")
			})
			oauth := OauthConfig{
				ClientID: "example-client", ClientSecret: "example-secret", TokenURL: "https://example.com/token",
				TokenTransport: func(stdhttp.RoundTripper) stdhttp.RoundTripper {
					return middlewares.RoundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
						credentialCalls++
						return nil, errors.New("unexpected OAuth token request")
					})
				},
			}
			aws := awssigv4.Config{Region: "us-east-1", Credentials: awssigv4.CredentialsProviderFunc(func(context.Context) (awssigv4.Credentials, error) {
				credentialCalls++
				return awssigv4.Credentials{}, errors.New("unexpected SigV4 credential request")
			})}
			var hosts []string
			client := capturingClient(&hosts)
			configure(client, bearer, oauth, aws)
			body := &closeRecordingBody{Reader: strings.NewReader("example payload")}
			req, err := stdhttp.NewRequest(stdhttp.MethodPost, "http://localhost/resource", body)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.RoundTrip(req)

			Expect(err).To(MatchError(ContainSubstring("BearerToken cannot be combined with " + conflictingMethod)))
			Expect(body.closes).To(Equal(1))
			Expect(credentialCalls).To(BeZero())
			Expect(hosts).To(BeEmpty())
		},
		Entry("bearer then OAuth", func(c *Client, bearer tokenSourceFunc, oauth OauthConfig, _ awssigv4.Config) {
			c.BearerToken(bearer).OAuth(oauth)
		}, "OAuth"),
		Entry("OAuth then bearer", func(c *Client, bearer tokenSourceFunc, oauth OauthConfig, _ awssigv4.Config) {
			c.OAuth(oauth).BearerToken(bearer)
		}, "OAuth"),
		Entry("bearer and SigV4", func(c *Client, bearer tokenSourceFunc, _ OauthConfig, aws awssigv4.Config) {
			c.BearerToken(bearer).AWSAuthSigV4(aws)
		}, "AWSAuthSigV4"),
	)
})
