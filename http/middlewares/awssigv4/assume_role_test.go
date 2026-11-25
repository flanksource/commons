package awssigv4

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AssumeRoleProvider", func() {
	const region = "eu-west-1"
	base := NewStaticCredentialsProvider(Credentials{AccessKeyID: "AKIDBASE", SecretAccessKey: "base-secret", SessionToken: "base-token"})

	options := func(endpoint string) AssumeRoleOptions {
		return AssumeRoleOptions{
			Region: region, Endpoint: endpoint, Base: base,
			RoleARN: "arn:aws:iam::123456789012:role/example", SessionName: "example-session",
			ExternalID: "example-external", Duration: 15 * time.Minute,
		}
	}

	It("sends a SigV4-signed AssumeRole query request and parses the credentials", func() {
		expires := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
		fake := newFakeSTS(func(int, url.Values) (int, string) {
			return 200, stsCredentialsXML("AssumeRole", "AKIDASSUMED", expires)
		})
		defer fake.Close()

		credentials, err := NewAssumeRoleProvider(options(fake.URL)).Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials).To(Equal(Credentials{AccessKeyID: "AKIDASSUMED", SecretAccessKey: "example-secret", SessionToken: "example-token", Expires: expires}))

		requests := fake.Requests()
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].Method).To(Equal(http.MethodPost))
		Expect(requests[0].ContentType).To(HavePrefix("application/x-www-form-urlencoded"))
		Expect(requests[0].Authorization).To(MatchRegexp(`^AWS4-HMAC-SHA256 Credential=AKIDBASE/\d{8}/eu-west-1/sts/aws4_request, SignedHeaders=.*, Signature=[0-9a-f]{64}$`))
		Expect(requests[0].SecurityToken).To(Equal("base-token"))
		Expect(requests[0].Form).To(Equal(url.Values{
			"Action": {"AssumeRole"}, "Version": {"2011-06-15"}, "RoleArn": {"arn:aws:iam::123456789012:role/example"},
			"RoleSessionName": {"example-session"}, "ExternalId": {"example-external"}, "DurationSeconds": {"900"},
		}))
	})

	It("surfaces the STS error code and message", func() {
		fake := newFakeSTS(func(int, url.Values) (int, string) { return 403, stsErrorXML })
		defer fake.Close()

		_, err := NewAssumeRoleProvider(options(fake.URL)).Retrieve(context.Background())
		var stsErr *STSError
		Expect(errors.As(err, &stsErr)).To(BeTrue())
		Expect(*stsErr).To(Equal(STSError{StatusCode: 403, Code: "AccessDenied", Message: "not authorized to assume role", RequestID: "req-denied"}))
		Expect(err.Error()).To(ContainSubstring("AccessDenied"))
	})

	It("fails on a success response without credentials and on non-XML failures", func() {
		for _, response := range []struct {
			status int
			body   string
		}{{200, `<AssumeRoleResponse><AssumeRoleResult></AssumeRoleResult></AssumeRoleResponse>`}, {502, "bad gateway"}} {
			fake := newFakeSTS(func(int, url.Values) (int, string) { return response.status, response.body })
			_, err := NewAssumeRoleProvider(options(fake.URL)).Retrieve(context.Background())
			Expect(err).To(HaveOccurred())
			fake.Close()
		}
	})

	It("rejects missing required options and failing base credentials", func() {
		for _, mutate := range []func(*AssumeRoleOptions){
			func(o *AssumeRoleOptions) { o.Region = "" }, func(o *AssumeRoleOptions) { o.Base = nil },
			func(o *AssumeRoleOptions) { o.RoleARN = "" }, func(o *AssumeRoleOptions) { o.SessionName = "" },
			func(o *AssumeRoleOptions) { o.Base = StaticCredentialsProvider{} },
		} {
			o := options("http://127.0.0.1:1")
			mutate(&o)
			_, err := NewAssumeRoleProvider(o).Retrieve(context.Background())
			Expect(err).To(HaveOccurred())
		}
	})

	It("refreshes through the credentials cache when the assumed credentials near expiry", func() {
		now := time.Now()
		fake := newFakeSTS(func(call int, _ url.Values) (int, string) {
			return 200, stsCredentialsXML("AssumeRole", map[int]string{1: "AKIDONE", 2: "AKIDTWO"}[call], now.Add(time.Hour))
		})
		defer fake.Close()
		cache := NewCredentialsCache(NewAssumeRoleProvider(options(fake.URL)))
		cache.now = func() time.Time { return now }

		first, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		again, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect([]string{first.AccessKeyID, again.AccessKeyID}).To(Equal([]string{"AKIDONE", "AKIDONE"}))
		Expect(fake.Requests()).To(HaveLen(1))

		now = now.Add(59 * time.Minute)
		refreshed, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(refreshed.AccessKeyID).To(Equal("AKIDTWO"))
	})
})
