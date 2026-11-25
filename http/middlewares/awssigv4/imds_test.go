package awssigv4

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IMDSProvider", func() {
	const token = "imds-session-token"

	BeforeEach(func() {
		GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "")
	})

	newIMDS := func(expires time.Time, tokenStatus int, credsStatus int) (*httptest.Server, *[]string) {
		var seen []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = append(seen, r.Method+" "+r.URL.Path)
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/latest/api/token":
				if r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(tokenStatus)
				_, _ = w.Write([]byte(token))
			case r.Header.Get("X-aws-ec2-metadata-token") != token:
				w.WriteHeader(http.StatusUnauthorized)
			case r.URL.Path == "/latest/meta-data/iam/security-credentials/":
				w.WriteHeader(credsStatus)
				_, _ = w.Write([]byte("example-role\n"))
			case r.URL.Path == "/latest/meta-data/iam/security-credentials/example-role":
				_, _ = fmt.Fprintf(w, `{"Code":"Success","AccessKeyId":"AKIDIMDS","SecretAccessKey":"imds-secret","Token":"imds-token","Expiration":"%s"}`, expires.UTC().Format(time.RFC3339))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		return server, &seen
	}

	It("performs the IMDSv2 token handshake then reads role credentials", func() {
		expires := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
		server, seen := newIMDS(expires, 200, 200)
		defer server.Close()

		credentials, err := NewIMDSProvider(IMDSOptions{Endpoint: server.URL}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials).To(Equal(Credentials{AccessKeyID: "AKIDIMDS", SecretAccessKey: "imds-secret", SessionToken: "imds-token", Expires: expires}))
		Expect(*seen).To(Equal([]string{"PUT /latest/api/token", "GET /latest/meta-data/iam/security-credentials/", "GET /latest/meta-data/iam/security-credentials/example-role"}))
	})

	It("uses AWS_EC2_METADATA_SERVICE_ENDPOINT when no endpoint is configured", func() {
		server, _ := newIMDS(time.Now().Add(time.Hour), 200, 200)
		defer server.Close()
		GinkgoT().Setenv("AWS_EC2_METADATA_SERVICE_ENDPOINT", server.URL)
		_, err := NewIMDSProvider(IMDSOptions{}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})

	It("fails when the token request fails, credentials are unauthorized, or IMDS is disabled", func() {
		failing, _ := newIMDS(time.Now().Add(time.Hour), 403, 200)
		defer failing.Close()
		_, err := NewIMDSProvider(IMDSOptions{Endpoint: failing.URL}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("403")))

		unauthorized, _ := newIMDS(time.Now().Add(time.Hour), 200, 401)
		defer unauthorized.Close()
		_, err = NewIMDSProvider(IMDSOptions{Endpoint: unauthorized.URL}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("401")))

		GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "true")
		_, err = NewIMDSProvider(IMDSOptions{Endpoint: failing.URL}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("AWS_EC2_METADATA_DISABLED")))
	})

	It("fails fast against an unreachable endpoint", func() {
		server := httptest.NewServer(http.NotFoundHandler())
		endpoint := server.URL
		server.Close()
		started := time.Now()
		_, err := NewIMDSProvider(IMDSOptions{Endpoint: endpoint}).Retrieve(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(time.Since(started)).To(BeNumerically("<", 5*time.Second))
	})
})
