package awssigv4

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContainerProvider", func() {
	clearContainerEnv := func() {
		for _, name := range []string{
			"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI",
			"AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", "AWS_CONTAINER_AUTHORIZATION_TOKEN",
			"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
			"AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_ROLE_ARN",
		} {
			GinkgoT().Setenv(name, "")
		}
	}

	type seenRequest struct{ Path, Authorization string }

	newContainerServer := func(expires time.Time, seen *[]seenRequest) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*seen = append(*seen, seenRequest{r.URL.Path, r.Header.Get("Authorization")})
			_, _ = fmt.Fprintf(w, `{"AccessKeyId":"AKIDCONTAINER","SecretAccessKey":"container-secret","Token":"container-token","Expiration":"%s"}`, expires.UTC().Format(time.RFC3339))
		}))
	}

	BeforeEach(clearContainerEnv)

	It("resolves the relative URI against the configured endpoint", func() {
		var seen []seenRequest
		server := newContainerServer(time.Now().Add(time.Hour), &seen)
		defer server.Close()
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/v2/credentials/example-id")

		credentials, err := NewContainerProvider(ContainerOptions{Endpoint: server.URL}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials.AccessKeyID).To(Equal("AKIDCONTAINER"))
		Expect(credentials.SessionToken).To(Equal("container-token"))
		Expect(seen).To(Equal([]seenRequest{{Path: "/v2/credentials/example-id"}}))
	})

	It("defaults the relative URI base to 169.254.170.2", func() {
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/creds")
		target, err := NewContainerProvider(ContainerOptions{}).targetURL()
		Expect(err).NotTo(HaveOccurred())
		Expect(target).To(Equal("http://169.254.170.2/creds"))
	})

	It("prefers the relative URI over the full URI", func() {
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/relative")
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "http://127.0.0.1:1/full")
		target, err := NewContainerProvider(ContainerOptions{}).targetURL()
		Expect(err).NotTo(HaveOccurred())
		Expect(target).To(Equal("http://169.254.170.2/relative"))
	})

	DescribeTable("full URI host validation",
		func(uri string, allowed bool) {
			GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", uri)
			_, err := NewContainerProvider(ContainerOptions{}).targetURL()
			if allowed {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("https any host", "https://creds.example.com/x", true),
		Entry("http loopback v4", "http://127.0.0.1:8080/x", true),
		Entry("http loopback v6", "http://[::1]:8080/x", true),
		Entry("http ECS host", "http://169.254.170.2/x", true),
		Entry("http EKS pod identity v4", "http://169.254.170.23/x", true),
		Entry("http EKS pod identity v6", "http://[fd00:ec2::23]/x", true),
		Entry("http non-loopback", "http://10.0.0.5/x", false),
		Entry("http hostname", "http://creds.example.com/x", false),
		Entry("unsupported scheme", "ftp://127.0.0.1/x", false),
	)

	It("rejects a non-loopback http full URI without sending a request", func() {
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "http://10.0.0.5/x")
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("AWS_CONTAINER_CREDENTIALS_FULL_URI")))
	})

	It("reports not configured when neither URI is set", func() {
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")))
	})

	It("sends the token file contents trimmed, re-read on each refresh, over the token env", func() {
		var seen []seenRequest
		server := newContainerServer(time.Now().Add(time.Hour), &seen)
		defer server.Close()
		tokenFile := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(tokenFile, []byte("file-token-1\n"), 0o600)).To(Succeed())
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL+"/pod")
		GinkgoT().Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", tokenFile)
		GinkgoT().Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN", "env-token")
		provider := NewContainerProvider(ContainerOptions{})

		_, err := provider.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(tokenFile, []byte("file-token-2"), 0o600)).To(Succeed())
		_, err = provider.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(seen).To(Equal([]seenRequest{{"/pod", "file-token-1"}, {"/pod", "file-token-2"}}))
	})

	It("falls back to the token env when no token file is set", func() {
		var seen []seenRequest
		server := newContainerServer(time.Now().Add(time.Hour), &seen)
		defer server.Close()
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL+"/pod")
		GinkgoT().Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN", "env-token")
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(seen).To(Equal([]seenRequest{{"/pod", "env-token"}}))
	})

	It("fails when the token file cannot be read", func() {
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "http://127.0.0.1:1/pod")
		GinkgoT().Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE", filepath.Join(GinkgoT().TempDir(), "missing"))
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("authorization token file")))
	})

	It("turns an error body into a typed error carrying the status", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"AccessDenied","message":"pod not associated"}`))
		}))
		defer server.Close()
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL)
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		var typed *ContainerError
		Expect(errors.As(err, &typed)).To(BeTrue())
		Expect(*typed).To(Equal(ContainerError{StatusCode: http.StatusForbidden, Code: "AccessDenied", Message: "pod not associated"}))
	})

	It("rejects an unparseable success body", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) }))
		defer server.Close()
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL)
		_, err := NewContainerProvider(ContainerOptions{}).Retrieve(context.Background())
		Expect(err).To(MatchError(ContainSubstring("parse container credentials")))
	})

	It("is refreshed through the cache only once the credentials near expiry", func() {
		var seen []seenRequest
		expires := time.Now().Add(time.Hour)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = append(seen, seenRequest{Path: r.URL.Path})
			_, _ = fmt.Fprintf(w, `{"AccessKeyId":"AKIDCONTAINER","SecretAccessKey":"s","Token":"t","Expiration":"%s"}`, expires.UTC().Format(time.RFC3339))
		}))
		defer server.Close()
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL)
		cache := NewCredentialsCache(NewContainerProvider(ContainerOptions{}))
		now := time.Now()
		cache.now = func() time.Time { return now }

		for range 2 {
			_, err := cache.Retrieve(context.Background())
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(seen).To(HaveLen(1))

		now = expires.Add(-30 * time.Second)
		expires = expires.Add(time.Hour)
		_, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(seen).To(HaveLen(2))
	})

	Describe("in DefaultChain", func() {
		It("is consulted before IMDS", func() {
			var seen []seenRequest
			server := newContainerServer(time.Now().Add(time.Hour), &seen)
			defer server.Close()
			imdsHits := 0
			imds := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { imdsHits++ }))
			defer imds.Close()
			GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL)

			credentials, err := DefaultChain("us-east-1", WithIMDSEndpoint(imds.URL)).Retrieve(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(credentials.AccessKeyID).To(Equal("AKIDCONTAINER"))
			Expect(imdsHits).To(BeZero())
		})

		It("is skipped when no container URI is set", func() {
			GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "true")
			_, err := DefaultChain("us-east-1").Retrieve(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("ContainerProvider"))
			Expect(err.Error()).To(ContainSubstring("IMDSProvider"))
		})
	})
})
