package awssigv4

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/flanksource/commons/http/middlewares"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS credential endpoint redirects", func() {
	var client *http.Client
	var redirectChecks int
	var paths []string
	var redirectPath string

	const endpoint = "https://example.com"

	BeforeEach(func() {
		redirectChecks = 0
		paths = nil
		client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				redirectChecks++
				return nil
			},
			Transport: middlewares.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.Body != nil {
					_ = req.Body.Close()
				}
				paths = append(paths, req.URL.Path)
				response := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Request: req}
				var body string
				switch req.URL.Path {
				case redirectPath:
					response.StatusCode = http.StatusTemporaryRedirect
					response.Header.Set("Location", "/next")
				case "/latest/api/token":
					body = "example-token"
				case imdsRolesPath:
					body = "example-role"
				default:
					return nil, errors.New("unexpected request to mock credential endpoint")
				}
				response.Body = io.NopCloser(strings.NewReader(body))
				return response, nil
			}),
		}
	})

	AfterEach(func() {
		Expect(redirectChecks).To(BeZero())
		Expect(client.CheckRedirect(nil, nil)).To(Succeed())
		Expect(redirectChecks).To(Equal(1))
	})

	It("rejects an STS redirect", func() {
		redirectPath = "/sts"
		call := stsCall{
			Action: "AssumeRoleWithWebIdentity", Region: "us-east-1", Endpoint: endpoint + redirectPath,
			Client: client, Params: url.Values{"WebIdentityToken": {"example-assertion"}},
		}
		_, err := call.do(context.Background())
		Expect(err).To(MatchError(ContainSubstring("HTTP 307")))
		Expect(paths).To(Equal([]string{redirectPath}))
	})

	DescribeTable("rejects an IMDSv2 redirect at any handshake step",
		func(expectedPaths []string) {
			GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "")
			redirectPath = expectedPaths[len(expectedPaths)-1]
			_, err := NewIMDSProvider(IMDSOptions{Endpoint: endpoint, HTTPClient: client}).Retrieve(context.Background())
			Expect(err).To(MatchError(ContainSubstring("HTTP 307")))
			Expect(paths).To(Equal(expectedPaths))
		},
		Entry("token", []string{"/latest/api/token"}),
		Entry("role name", []string{"/latest/api/token", imdsRolesPath}),
		Entry("credentials", []string{"/latest/api/token", imdsRolesPath, imdsRolesPath + "example-role"}),
	)
})
