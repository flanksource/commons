package awssigv4

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebIdentityProvider", func() {
	const roleARN = "arn:aws:iam::123456789012:role/example-irsa"

	writeToken := func(token string) string {
		path := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(path, []byte(token+"\n"), 0o600)).To(Succeed())
		return path
	}

	It("calls AssumeRoleWithWebIdentity unsigned using explicit options", func() {
		expires := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
		fake := newFakeSTS(func(int, url.Values) (int, string) {
			return 200, stsCredentialsXML("AssumeRoleWithWebIdentity", "AKIDWEB", expires)
		})
		defer fake.Close()

		credentials, err := NewWebIdentityProvider(WebIdentityOptions{
			Region: "us-east-2", Endpoint: fake.URL, TokenFile: writeToken("example-jwt"), RoleARN: roleARN, SessionName: "example-session",
		}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials).To(Equal(Credentials{AccessKeyID: "AKIDWEB", SecretAccessKey: "example-secret", SessionToken: "example-token", Expires: expires}))

		requests := fake.Requests()
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].Authorization).To(BeEmpty())
		Expect(requests[0].Form).To(Equal(url.Values{
			"Action": {"AssumeRoleWithWebIdentity"}, "Version": {"2011-06-15"}, "RoleArn": {roleARN},
			"RoleSessionName": {"example-session"}, "WebIdentityToken": {"example-jwt"},
		}))
	})

	It("reads the token file, role and session name from the environment", func() {
		fake := newFakeSTS(func(int, url.Values) (int, string) {
			return 200, stsCredentialsXML("AssumeRoleWithWebIdentity", "AKIDENV", time.Now().Add(time.Hour))
		})
		defer fake.Close()
		GinkgoT().Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", writeToken("env-jwt"))
		GinkgoT().Setenv("AWS_ROLE_ARN", roleARN)
		GinkgoT().Setenv("AWS_ROLE_SESSION_NAME", "env-session")

		_, err := NewWebIdentityProvider(WebIdentityOptions{Region: "us-east-2", Endpoint: fake.URL}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		form := fake.Requests()[0].Form
		Expect([]string{form.Get("RoleArn"), form.Get("RoleSessionName"), form.Get("WebIdentityToken")}).To(Equal([]string{roleARN, "env-session", "env-jwt"}))
	})

	It("surfaces STS errors and rejects missing configuration", func() {
		fake := newFakeSTS(func(int, url.Values) (int, string) { return 400, stsErrorXML })
		defer fake.Close()
		_, err := NewWebIdentityProvider(WebIdentityOptions{Region: "us-east-2", Endpoint: fake.URL, TokenFile: writeToken("t"), RoleARN: roleARN}).Retrieve(context.Background())
		var stsErr *STSError
		Expect(errors.As(err, &stsErr)).To(BeTrue())
		Expect(stsErr.Code).To(Equal("AccessDenied"))

		GinkgoT().Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
		GinkgoT().Setenv("AWS_ROLE_ARN", "")
		for _, o := range []WebIdentityOptions{
			{Region: "us-east-2"},
			{Region: "us-east-2", RoleARN: roleARN},
			{Region: "us-east-2", RoleARN: roleARN, TokenFile: filepath.Join(GinkgoT().TempDir(), "missing")},
			{RoleARN: roleARN, TokenFile: writeToken("t")},
		} {
			_, err := NewWebIdentityProvider(o).Retrieve(context.Background())
			Expect(err).To(HaveOccurred())
		}
	})
})
