package sdkcompat_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSDKCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SigV4 AWS SDK Compatibility")
}

type matrixCase struct{ method, service, query, path, body, token string }

var _ = Describe("real AWS SDK cross-check", func() {
	var cases []TableEntry
	for _, method := range []string{"GET", "POST"} {
		for _, service := range []string{"s3", "execute-api", "sts"} {
			for _, query := range []string{"", "z=last&a=first&a=second&empty=&novalue", "q=space+and%20plus%2B&other=%21%2F", "key%2D=one&key=two"} {
				for _, path := range []string{"/", "/文書/ሴ", "/already%2Fencoded/a%20b"} {
					for _, body := range []string{"", `{"message":"hello"}`} {
						for _, token := range []string{"", "example-session-token"} {
							value := matrixCase{method, service, query, path, body, token}
							cases = append(cases, Entry(fmt.Sprintf("%s %s %s ?%s body=%t session=%t", method, service, path, query, body != "", token != ""), value))
						}
					}
				}
			}
		}
	}
	DescribeTable("produces identical signed request headers", func(value matrixCase) {
		request, err := http.NewRequest(value.method, "https://example.amazonaws.com:443"+value.path+"?"+value.query, strings.NewReader(value.body))
		Expect(err).NotTo(HaveOccurred())
		request.Header = http.Header{"Content-Type": {"application/json"}, "X-Example": {" a  b  ", "c"}, "User-Agent": {"example-client"}, "Expect": {"100-continue"}, "X-Amzn-Trace-Id": {"example-trace"}}
		hash := sha256.Sum256([]byte(value.body))
		payloadHash := hex.EncodeToString(hash[:])
		if value.service == "s3" {
			request.Header.Set("X-Amz-Content-Sha256", payloadHash)
		}
		ours := request.Clone(context.Background())
		expected := request.Clone(context.Background())
		when := time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)
		_, err = awssigv4.Sign(ours, awssigv4.SignOptions{Credentials: awssigv4.Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "example-secret", SessionToken: value.token}, Region: "us-east-1", Service: value.service, Time: when, PayloadHash: payloadHash})
		Expect(err).NotTo(HaveOccurred())
		err = v4.NewSigner(func(options *v4.SignerOptions) { options.DisableURIPathEscaping = value.service == "s3" }).SignHTTP(context.Background(), aws.Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "example-secret", SessionToken: value.token}, expected, payloadHash, value.service, "us-east-1", when)
		Expect(err).NotTo(HaveOccurred())
		Expect(ours.Header).To(Equal(expected.Header))
		Expect(ours.Host).To(Equal(expected.Host))
		Expect(ours.URL.RawQuery).To(Equal(expected.URL.RawQuery))
	}, cases)
})
