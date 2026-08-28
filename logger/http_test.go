package logger

import (
	"bytes"
	"strings"

	"github.com/flanksource/commons/properties"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP response body log limit", func() {
	AfterEach(func() {
		properties.Set(HTTPLogResponseBodyLengthProperty, "")
	})

	It("defaults to four KiB", func() {
		Expect(HTTPLogResponseBodyLength(0)).To(Equal(int64(4 * 1024)))
	})

	It("uses the property instead of a trace fallback", func() {
		properties.Set(HTTPLogResponseBodyLengthProperty, 9_999_999)
		Expect(HTTPLogResponseBodyLength(4 * 1024)).To(Equal(int64(9_999_999)))
	})

	It("redacts an incomplete JSON prefix", func() {
		const secret = "supersecret"
		var output bytes.Buffer
		Expect(redactedJSONFormatter{}.Format(&output, []byte(`{"password":"supersecret"`))).To(Succeed())
		Expect(output.String()).ToNot(ContainSubstring(secret))
		Expect(strings.TrimSpace(output.String())).ToNot(BeEmpty())
	})
})
