package sdkcompat_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type seenRequest struct {
	Form          url.Values
	ContentType   string
	Authorization string
}

// recordingSTS answers every request with the given Query API action's credentials document and records the request.
func recordingSTS(action string, expires time.Time) (*httptest.Server, *seenRequest) {
	seen := &seenRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		seen.Form, err = url.ParseQuery(string(body))
		Expect(err).NotTo(HaveOccurred())
		seen.ContentType, seen.Authorization = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`<` + action + `Response xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><` + action + `Result><Credentials><AccessKeyId>AKIDRESULT</AccessKeyId><SecretAccessKey>example-secret</SecretAccessKey><SessionToken>example-token</SessionToken><Expiration>` + expires.UTC().Format(time.RFC3339) + `</Expiration></Credentials></` + action + `Result></` + action + `Response>`))
	}))
	return server, seen
}

var scope = regexp.MustCompile(`Credential=AKIDBASE/\d{8}/us-west-2/sts/aws4_request`)

var _ = Describe("STS providers against the real AWS SDK", func() {
	const roleARN = "arn:aws:iam::123456789012:role/example"
	expires := time.Now().Add(time.Hour).Truncate(time.Second)
	base := awssigv4.NewStaticCredentialsProvider(awssigv4.Credentials{AccessKeyID: "AKIDBASE", SecretAccessKey: "base-secret"})

	sdkClient := func(endpoint string, provider aws.CredentialsProvider) *sts.Client {
		return sts.New(sts.Options{Region: "us-west-2", BaseEndpoint: aws.String(endpoint), Credentials: provider, HTTPClient: http.DefaultClient})
	}

	It("sends the same AssumeRole body, content type and credential scope as the SDK and parses the same credentials", func() {
		server, sdkSeen := recordingSTS("AssumeRole", expires)
		defer server.Close()
		sdkOut, err := sdkClient(server.URL, credentials.NewStaticCredentialsProvider("AKIDBASE", "base-secret", "")).AssumeRole(context.Background(), &sts.AssumeRoleInput{
			RoleArn: aws.String(roleARN), RoleSessionName: aws.String("example-session"), ExternalId: aws.String("example-external"), DurationSeconds: aws.Int32(900),
		})
		Expect(err).NotTo(HaveOccurred())

		oursServer, oursSeen := recordingSTS("AssumeRole", expires)
		defer oursServer.Close()
		ours, err := awssigv4.NewAssumeRoleProvider(awssigv4.AssumeRoleOptions{
			Region: "us-west-2", Endpoint: oursServer.URL, Base: base, RoleARN: roleARN, SessionName: "example-session", ExternalID: "example-external", Duration: 15 * time.Minute,
		}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(oursSeen.Form).To(Equal(sdkSeen.Form))
		Expect(oursSeen.ContentType).To(Equal(sdkSeen.ContentType))
		Expect(sdkSeen.Authorization).To(MatchRegexp(scope.String()))
		Expect(oursSeen.Authorization).To(MatchRegexp(scope.String()))
		Expect(ours).To(Equal(awssigv4.Credentials{AccessKeyID: *sdkOut.Credentials.AccessKeyId, SecretAccessKey: *sdkOut.Credentials.SecretAccessKey, SessionToken: *sdkOut.Credentials.SessionToken, Expires: *sdkOut.Credentials.Expiration}))
	})

	It("sends the same unsigned AssumeRoleWithWebIdentity body as the SDK and parses the same credentials", func() {
		server, sdkSeen := recordingSTS("AssumeRoleWithWebIdentity", expires)
		defer server.Close()
		sdkOut, err := sdkClient(server.URL, aws.AnonymousCredentials{}).AssumeRoleWithWebIdentity(context.Background(), &sts.AssumeRoleWithWebIdentityInput{
			RoleArn: aws.String(roleARN), RoleSessionName: aws.String("example-session"), WebIdentityToken: aws.String("example-jwt"),
		})
		Expect(err).NotTo(HaveOccurred())

		tokenFile := filepath.Join(GinkgoT().TempDir(), "token")
		Expect(os.WriteFile(tokenFile, []byte("example-jwt"), 0o600)).To(Succeed())
		oursServer, oursSeen := recordingSTS("AssumeRoleWithWebIdentity", expires)
		defer oursServer.Close()
		ours, err := awssigv4.NewWebIdentityProvider(awssigv4.WebIdentityOptions{
			Region: "us-west-2", Endpoint: oursServer.URL, TokenFile: tokenFile, RoleARN: roleARN, SessionName: "example-session",
		}).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(oursSeen.Form).To(Equal(sdkSeen.Form))
		Expect(oursSeen.ContentType).To(Equal(sdkSeen.ContentType))
		Expect(oursSeen.Authorization).To(BeEmpty())
		Expect(sdkSeen.Authorization).To(BeEmpty())
		Expect(ours).To(Equal(awssigv4.Credentials{AccessKeyID: *sdkOut.Credentials.AccessKeyId, SecretAccessKey: *sdkOut.Credentials.SecretAccessKey, SessionToken: *sdkOut.Credentials.SessionToken, Expires: *sdkOut.Credentials.Expiration}))
	})
})
