package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClientCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP client compatibility")
}

var _ = Describe("current commons HTTP APIs", func() {
	It("preserves expiring SDK credentials and retrieval errors", func() {
		expires := time.Now().Add(time.Hour)
		provider := sdkCredentialsProvider(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-token", CanExpire: true, Expires: expires}, nil
		}))
		value, err := provider.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(awssigv4.Credentials{AccessKeyID: "example-key", SecretAccessKey: "example-secret", SessionToken: "example-token", Expires: expires}))
		failure := errors.New("credential retrieval failed")
		provider = sdkCredentialsProvider(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) { return aws.Credentials{}, failure }))
		_, err = provider.Retrieve(context.Background())
		Expect(err).To(MatchError(failure))
	})
	It("returns missing credential errors instead of exiting", func() {
		GinkgoT().Setenv("AWS_ACCESS_KEY_ID", "")
		GinkgoT().Setenv("AWS_SECRET_ACCESS_KEY", "")
		GinkgoT().Setenv("AWS_PROFILE", "")
		GinkgoT().Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
		GinkgoT().Setenv("AWS_ROLE_ARN", "")
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "")
		GinkgoT().Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "")
		GinkgoT().Setenv("AWS_CONFIG_FILE", filepath.Join(GinkgoT().TempDir(), "config"))
		GinkgoT().Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(GinkgoT().TempDir(), "credentials"))
		GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "true")
		flagAWSSigV4, flagAWSRegion = true, "us-east-1"
		DeferCleanup(func() { flagAWSSigV4, flagAWSRegion = false, "" })
		_, _, err := buildClient()
		Expect(err).To(MatchError(ContainSubstring("retrieving AWS credentials")))
	})
	It("obtains an OAuth token and authenticates the resource request", func() {
		var authorization string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"example-token","token_type":"Bearer","expires_in":3600}`))
				return
			}
			authorization = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusNoContent)
		}))
		DeferCleanup(server.Close)
		flagOAuthClientID, flagOAuthSecret, flagOAuthTokenURL = "example-client", "example-secret", server.URL+"/token"
		DeferCleanup(func() { flagOAuthClientID, flagOAuthSecret, flagOAuthTokenURL = "", "", "" })
		client, _, err := buildClient()
		Expect(err).NotTo(HaveOccurred())
		resp, err := client.R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Body.Close()).To(Succeed())
		Expect(authorization).To(Equal("Bearer example-token"))
	})
	It("rejects NTLM before sending any request, even without a username", func() {
		flagNTLM = true
		DeferCleanup(func() { flagNTLM = false })
		Expect(run(rootCmd, []string{"http://127.0.0.1:1"})).To(MatchError(ContainSubstring("NTLM authentication is no longer supported")))
	})
	It("signs with the SDK environment credential chain and session token", func() {
		GinkgoT().Setenv("AWS_ACCESS_KEY_ID", "example-access-key")
		GinkgoT().Setenv("AWS_SECRET_ACCESS_KEY", "example-secret-key")
		GinkgoT().Setenv("AWS_SESSION_TOKEN", "example-session-token")
		GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "true")
		flagAWSSigV4, flagAWSRegion, flagAWSService = true, "us-east-1", "execute-api"
		DeferCleanup(func() { flagAWSSigV4, flagAWSRegion, flagAWSService = false, "", "" })
		var authorization, token string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization, token = r.Header.Get("Authorization"), r.Header.Get("X-Amz-Security-Token")
			w.WriteHeader(http.StatusNoContent)
		}))
		DeferCleanup(server.Close)
		client, _, err := buildClient()
		Expect(err).NotTo(HaveOccurred())
		resp, err := client.R(context.Background()).Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Body.Close()).To(Succeed())
		Expect(authorization).To(HavePrefix("AWS4-HMAC-SHA256 Credential=example-access-key/"))
		Expect(token).To(Equal("example-session-token"))
	})
})
