package har_test

import (
	"net/http"
	"net/url"

	"github.com/flanksource/commons/har"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HAR request redaction", func() {
	It("masks credential headers and token, code and sig query values", func() {
		address, err := url.Parse("https://example.test/?token=token-secret&code=code-secret&sig=signature-secret&view=public")
		Expect(err).NotTo(HaveOccurred())
		headers := http.Header{"Authorization": {"Bearer authorization-secret"}, "Proxy-Authorization": {"Bearer proxy-secret"}, "Cookie": {"session=cookie-secret"}}
		request := har.RedactRequest(address, headers, har.HARConfig{})
		for _, secret := range []string{"token-secret", "code-secret", "signature-secret", "authorization-secret", "proxy-secret", "cookie-secret"} {
			Expect(request.URL).NotTo(ContainSubstring(secret))
			Expect(request.Headers).NotTo(ContainElement(HaveField("Value", ContainSubstring(secret))))
			Expect(request.QueryString).NotTo(ContainElement(HaveField("Value", ContainSubstring(secret))))
		}
		Expect(request.URL).To(ContainSubstring("view=public"))
	})
})
