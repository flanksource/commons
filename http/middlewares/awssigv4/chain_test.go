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

var _ = Describe("Chain", func() {
	good := func(key string) CredentialsProvider {
		return NewStaticCredentialsProvider(Credentials{AccessKeyID: key, SecretAccessKey: "example-secret"})
	}
	failing := func(err error) CredentialsProvider {
		return CredentialsProviderFunc(func(context.Context) (Credentials, error) { return Credentials{}, err })
	}

	It("returns the first successful provider without consulting later ones", func() {
		called := false
		later := CredentialsProviderFunc(func(context.Context) (Credentials, error) { called = true; return Credentials{}, nil })
		credentials, err := Chain(failing(errors.New("first failed")), good("AKIDSECOND"), later).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials.AccessKeyID).To(Equal("AKIDSECOND"))
		Expect(called).To(BeFalse())
	})

	It("joins every provider error naming each provider when all fail", func() {
		GinkgoT().Setenv("AWS_ACCESS_KEY_ID", "")
		GinkgoT().Setenv("AWS_SECRET_ACCESS_KEY", "")
		errA, errB := errors.New("env missing"), errors.New("imds unreachable")
		_, err := Chain(failing(errA), EnvironmentProvider{}, failing(errB)).Retrieve(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, errA)).To(BeTrue())
		Expect(errors.Is(err, errB)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("EnvironmentProvider"))
		Expect(err.Error()).To(ContainSubstring("chain[0]"))
		Expect(err.Error()).To(ContainSubstring("chain[2]"))
	})

	It("rejects an empty chain", func() {
		_, err := Chain().Retrieve(context.Background())
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("DefaultChain", func() {
	clearEnv := func() {
		for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_ROLE_ARN", "AWS_ROLE_SESSION_NAME"} {
			GinkgoT().Setenv(name, "")
		}
	}

	It("prefers environment credentials", func() {
		clearEnv()
		GinkgoT().Setenv("AWS_ACCESS_KEY_ID", "AKIDENVIRON")
		GinkgoT().Setenv("AWS_SECRET_ACCESS_KEY", "example-secret")
		credentials, err := DefaultChain("us-east-1").Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials.AccessKeyID).To(Equal("AKIDENVIRON"))
	})

	It("falls through to web identity when configured by environment", func() {
		clearEnv()
		fake := newFakeSTS(func(int, url.Values) (int, string) {
			return 200, stsCredentialsXML("AssumeRoleWithWebIdentity", "AKIDWEBCHAIN", time.Now().Add(time.Hour))
		})
		defer fake.Close()
		tokenFile := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(tokenFile, []byte("jwt"), 0o600)).To(Succeed())
		GinkgoT().Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)
		GinkgoT().Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/example")

		credentials, err := DefaultChain("us-east-1", WithSTSEndpoint(fake.URL)).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials.AccessKeyID).To(Equal("AKIDWEBCHAIN"))
	})

	It("falls through to IMDS and reports every failure when nothing is available", func() {
		clearEnv()
		GinkgoT().Setenv("AWS_EC2_METADATA_DISABLED", "true")
		_, err := DefaultChain("us-east-1").Retrieve(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("EnvironmentProvider"))
		Expect(err.Error()).To(ContainSubstring("IMDSProvider"))
		Expect(err.Error()).NotTo(ContainSubstring("WebIdentityProvider"))
	})
})
