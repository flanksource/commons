package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	commonscontext "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPLogger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Logger Suite")
}

var _ = Describe("JSON response body logging limits", func() {
	It("logs a bounded prefix, marks truncation, and restores the response", func() {
		const (
			bodyLimit = 30
			body      = "value=ab&password=supersecret-remaining"
			secret    = "supersecret"
		)
		properties.Set("http.log.response.body.length", bodyLimit)
		properties.Set("log.json", true)
		DeferCleanup(func() { properties.Set("http.log.response.body.length", "") })
		DeferCleanup(func() { properties.Set("log.json", "") })

		var output bytes.Buffer
		log := logger.NewWithWriter(&output)
		requestContext := commonscontext.NewContext(context.Background(), commonscontext.WithLogger(log))
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, "https://example.com", nil)
		Expect(err).ToNot(HaveOccurred())

		response, err := jsonLogger(TraceConfig{Response: true, MaxBodyLength: 4096}, nil, RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: int64(len(body)),
				Header:        http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
				Body:          io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}), request)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(response.Body.Close)

		var record map[string]any
		Expect(json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record)).To(Succeed())
		Expect(record).To(HaveKeyWithValue("responseBodyTruncated", true))
		Expect(record["responseBody"]).To(BeAssignableToTypeOf(""))
		Expect(record["responseBody"].(string)).To(ContainSubstring("value=ab"))
		Expect(record["responseBody"].(string)).ToNot(ContainSubstring(secret))

		responseBody, err := io.ReadAll(response.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(responseBody)).To(Equal(body))
	})

	It("redacts secrets in a truncated error response", func() {
		const (
			bodyLimit = 40
			secret    = "supersecret"
			body      = `{"password":"supersecret","padding":"xxxxxxxxxxxxxxxxxxxxxxxx"}`
		)
		response := &http.Response{Body: io.NopCloser(bytes.NewBufferString(body))}

		logged := readErrorBody(response, bodyLimit)

		Expect(logged).To(ContainSubstring("body truncated after 40 bytes"))
		Expect(logged).ToNot(ContainSubstring(secret))
		responseBody, err := io.ReadAll(response.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(responseBody)).To(Equal(body))
	})
})
