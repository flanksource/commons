package awssigv4

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSigV4(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AWS SigV4 Suite")
}

var _ = Describe("official AWS SigV4 vectors", func() {
	const root = "testdata/aws-sig-v4-test-suite"
	var vectors []TableEntry
	err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(filename, ".req") {
			vectors = append(vectors, Entry(strings.TrimSuffix(strings.TrimPrefix(filename, root+"/"), ".req"), filename))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	if len(vectors) == 0 {
		panic("official AWS SigV4 vectors are missing")
	}
	DescribeTable("matches every canonical request, string to sign and authorization", func(filename string) {
		request, body := vectorRequest(filename)
		hash := sha256.Sum256([]byte(body))
		signingTime, err := time.Parse("20060102T150405Z", request.Header.Get("X-Amz-Date"))
		Expect(err).NotTo(HaveOccurred())
		options := SignOptions{
			Credentials: Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", SessionToken: request.Header.Get("X-Amz-Security-Token")},
			Region:      "us-east-1", Service: "service", Time: signingTime,
			PayloadHash: hex.EncodeToString(hash[:]), DisableURIPathEscaping: true,
			DisableSessionToken: strings.Contains(filename, "post-sts-header-after"),
		}
		if strings.Contains(filename, "get-vanilla-with-session-token") {
			options.Credentials.SessionToken = "example-session-token"
		}
		request.Header.Del("X-Amz-Security-Token")
		signature, err := Sign(request, options)
		Expect(err).NotTo(HaveOccurred())
		prefix := strings.TrimSuffix(filename, ".req")
		for suffix, actual := range map[string]string{".creq": signature.CanonicalRequest, ".sts": signature.StringToSign, ".authz": signature.Authorization} {
			expected, err := os.ReadFile(prefix + suffix)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(strings.TrimSuffix(string(expected), "\n")), suffix)
		}
		Expect(request.Header.Get("Authorization")).To(Equal(signature.Authorization))
	}, vectors)
})

func vectorRequest(filename string) (*http.Request, string) {
	data, err := os.ReadFile(filename)
	Expect(err).NotTo(HaveOccurred())
	headers, body, _ := strings.Cut(string(data), "\n\n")
	lines := strings.Split(headers, "\n")
	method, target, ok := strings.Cut(lines[0], " ")
	Expect(ok).To(BeTrue())
	target = target[:strings.LastIndex(target, " HTTP/")]
	header := make(http.Header)
	var previous string
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			values := header.Values(previous)
			values[len(values)-1] += " " + strings.TrimSpace(line)
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		Expect(ok).To(BeTrue())
		header.Add(key, strings.TrimSpace(value))
		previous = key
	}
	request, err := http.NewRequest(method, "https://"+header.Get("Host")+target, strings.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	request.Host = header.Get("Host")
	header.Del("Host")
	request.Header = header
	request.ContentLength = 0
	if value := header.Get("Content-Length"); value != "" {
		request.ContentLength, err = strconv.ParseInt(value, 10, 64)
		Expect(err).NotTo(HaveOccurred())
	}
	return request, body
}
