package http

import (
	"context"
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
			initial := start
			if endpoint != "" {
				initial = "http://" + endpoint + "/start"
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
				Expect(attempts).To(Equal([]string{initial, destination}))
			} else {
				Expect(err).To(MatchError(ContainSubstring("send credentials over cleartext HTTP")))
				Expect(attempts).To(Equal([]string{initial}))
			}
		},
		Entry("SigV4 to remote HTTP", "sigv4", "", "http://example.com/next", false),
		Entry("SigV4 to another remote HTTP host", "sigv4", "", "http://example.net/next", false),
		Entry("SigV4 to loopback HTTP", "sigv4", "", "http://[::1]/next", true),
		Entry("SigV4 to remote HTTPS", "sigv4", "", "https://example.com/next", true),
		Entry("SigV4 redirect rewritten by AWSEndpoint", "sigv4", "example.net:8080", "https://example.com/next", false),
		Entry("basic auth to remote HTTP", "basic", "", "http://example.com/next", false),
		Entry("no auth to remote HTTP", "", "", "http://example.com/next", true),
	)
})
