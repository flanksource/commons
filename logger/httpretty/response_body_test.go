package httpretty

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPPretty(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Pretty Suite")
}

var _ = Describe("server response body logging limits", func() {
	It("logs a bounded prefix and sends the full response", func() {
		const (
			bodyLimit = 8
			body      = "abcdefgh-remaining"
		)
		var output bytes.Buffer
		log := &Logger{ResponseBody: true, MaxResponseBody: bodyLimit}
		log.SetOutput(&output)
		handler := log.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "https://example.com", nil))

		Expect(response.Body.String()).To(Equal(body))
		Expect(output.String()).To(ContainSubstring(body[:bodyLimit]))
		Expect(output.String()).To(ContainSubstring("body truncated after 8 bytes"))
		Expect(output.String()).ToNot(ContainSubstring(body[bodyLimit:]))
	})
})
