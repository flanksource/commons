package http

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("authenticated redirects", func() {
	DescribeTable("enforces the redirect transport policy before sending a second request",
		func(auth, endpoint, destination string, allowed bool) {
			client := NewClient()
			switch auth {
			case "basic":
				client.Auth("example-user", "example-password")
			case "sigv4":
				client.AWSAuthSigV4(awssigv4.Config{
					Region: "us-east-1",
					Credentials: awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{
						AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-token",
					}),
				}).AWSService("execute-api").AWSEndpoint(endpoint)
			}
			const start = "https://example.com/start"
			initial, second := start, destination
			if endpoint != "" {
				initial = "http://" + endpoint + "/start"
				second = "http://" + endpoint + "/next"
			}
			var attempts []string
			client.Use(func(stdhttp.RoundTripper) stdhttp.RoundTripper {
				return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
					attempts = append(attempts, req.URL.String())
					resp := &stdhttp.Response{StatusCode: stdhttp.StatusNoContent, Header: stdhttp.Header{}, Body: io.NopCloser(strings.NewReader(""))}
					if req.URL.Path == "/start" {
						resp.StatusCode = stdhttp.StatusFound
						resp.Header.Set("Location", destination)
					}
					return resp, nil
				})
			})

			resp, err := client.R(context.Background()).Get(start)

			if allowed {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Body.Close()).To(Succeed())
				Expect(attempts).To(Equal([]string{initial, second}))
			} else {
				Expect(err).To(MatchError(ContainSubstring("send credentials over cleartext HTTP")))
				Expect(attempts).To(Equal([]string{initial}))
			}
		},
		Entry("SigV4 to remote HTTP", "sigv4", "", "http://example.com/next", false),
		Entry("SigV4 to another remote HTTP host", "sigv4", "", "http://example.net/next", false),
		Entry("SigV4 to loopback HTTP", "sigv4", "", "http://[::1]/next", true),
		Entry("SigV4 to remote HTTPS", "sigv4", "", "https://example.com/next", true),
		Entry("SigV4 redirect rewritten onto a loopback AWSEndpoint", "sigv4", "127.0.0.1:4566", "https://example.com/next", true),
		Entry("basic auth to remote HTTP", "basic", "", "http://example.com/next", false),
		Entry("no auth to remote HTTP", "", "", "http://example.com/next", true),
	)
})

var _ = Describe("credential transport on the first request", func() {
	DescribeTable("refuses cleartext credentials to a remote host before fetching them",
		func(auth, target, endpoint string, allowed bool) {
			var credentialCalls int
			client := NewClient()
			switch auth {
			case "sigv4":
				client.AWSAuthSigV4(awssigv4.Config{Region: "us-east-1", Credentials: awssigv4.CredentialsProviderFunc(func(context.Context) (awssigv4.Credentials, error) {
					credentialCalls++
					return awssigv4.Credentials{AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-token"}, nil
				})}).AWSService("execute-api").AWSEndpoint(endpoint)
			case "oauth":
				client.OAuth(OauthConfig{
					ClientID: "example-client", ClientSecret: "example-secret", TokenURL: "https://example.com/token",
					TokenTransport: func(stdhttp.RoundTripper) stdhttp.RoundTripper {
						return middlewares.RoundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
							credentialCalls++
							return nil, errors.New("token endpoint reached")
						})
					},
				})
			}
			var sent []string
			client.Use(func(stdhttp.RoundTripper) stdhttp.RoundTripper {
				return middlewares.RoundTripperFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
					sent = append(sent, req.URL.String())
					return &stdhttp.Response{StatusCode: stdhttp.StatusNoContent, Header: stdhttp.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
				})
			})

			resp, err := client.R(context.Background()).Get(target)

			if allowed {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Body.Close()).To(Succeed())
				Expect(credentialCalls).To(Equal(1))
				Expect(sent).To(HaveLen(1))
				return
			}
			Expect(err).To(MatchError(ContainSubstring("sending credentials requires HTTPS except on loopback HTTP endpoints")))
			Expect(credentialCalls).To(BeZero())
			Expect(sent).To(BeEmpty())
		},
		Entry("SigV4 with a remote AWSEndpoint", "sigv4", "https://example.com/resource", "example.net:8080", false),
		Entry("SigV4 with a loopback AWSEndpoint", "sigv4", "https://example.com/resource", "localhost:4566", true),
		Entry("SigV4 to a remote HTTP URL", "sigv4", "http://example.com/resource", "", false),
		Entry("OAuth to a remote HTTP URL", "oauth", "http://example.com/resource", "", false),
	)
})
