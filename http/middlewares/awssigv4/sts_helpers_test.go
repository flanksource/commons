package awssigv4

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	. "github.com/onsi/gomega"
)

type capturedSTSRequest struct {
	Method        string
	Path          string
	Authorization string
	SecurityToken string
	ContentType   string
	Form          url.Values
}

type fakeSTS struct {
	*httptest.Server
	mu       sync.Mutex
	requests []capturedSTSRequest
}

func (f *fakeSTS) Requests() []capturedSTSRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]capturedSTSRequest(nil), f.requests...)
}

func stsCredentialsXML(action, accessKey string, expires time.Time) string {
	return fmt.Sprintf(`<%[1]sResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><%[1]sResult><Credentials><AccessKeyId>%[2]s</AccessKeyId><SecretAccessKey>example-secret</SecretAccessKey><SessionToken>example-token</SessionToken><Expiration>%[3]s</Expiration></Credentials></%[1]sResult><ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata></%[1]sResponse>`,
		action, accessKey, expires.UTC().Format(time.RFC3339))
}

const stsErrorXML = `<ErrorResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><Error><Type>Sender</Type><Code>AccessDenied</Code><Message>not authorized to assume role</Message></Error><RequestId>req-denied</RequestId></ErrorResponse>`

func newFakeSTS(respond func(call int, form url.Values) (int, string)) *fakeSTS {
	fake := &fakeSTS{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		form, err := url.ParseQuery(string(body))
		Expect(err).NotTo(HaveOccurred())
		fake.mu.Lock()
		fake.requests = append(fake.requests, capturedSTSRequest{
			Method: r.Method, Path: r.URL.Path, Authorization: r.Header.Get("Authorization"),
			SecurityToken: r.Header.Get("X-Amz-Security-Token"), ContentType: r.Header.Get("Content-Type"), Form: form,
		})
		call := len(fake.requests)
		fake.mu.Unlock()
		status, payload := respond(call, form)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(payload))
	}))
	return fake
}
