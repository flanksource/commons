package properties

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("byte sizes", func() {
	DescribeTable("parses the documented syntax",
		func(value string, expected int64) {
			parsed, err := ParseBytes(value)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsed).To(Equal(expected))
		},
		Entry("plain bytes", "1048576", int64(1048576)),
		Entry("explicit B suffix", "512B", int64(512)),
		Entry("IEC mebibytes", "1MiB", int64(1024*1024)),
		Entry("IEC mebibytes lower case", "1mib", int64(1024*1024)),
		Entry("IEC short form", "1Mi", int64(1024*1024)),
		Entry("IEC kibibytes", "64KiB", int64(64*1024)),
		Entry("IEC gibibytes", "2GiB", int64(2*1024*1024*1024)),
		Entry("SI megabytes", "1MB", int64(1000*1000)),
		Entry("SI kilobytes", "4KB", int64(4000)),
		Entry("separating whitespace", " 1 MiB ", int64(1024*1024)),
		Entry("zero", "0", int64(0)),
		Entry("negative", "-1", int64(-1)),
	)

	DescribeTable("rejects values it cannot parse",
		func(value string) {
			_, err := ParseBytes(value)
			Expect(err).To(HaveOccurred())
		},
		Entry("empty", ""),
		Entry("unit only", "MiB"),
		Entry("unknown unit", "1PB"),
		Entry("trailing junk", "1MiBB"),
		Entry("fractional", "1.5MiB"),
		Entry("overflow", "9999999999TiB"),
	)

	Describe("Properties.Bytes", func() {
		var store *Properties

		BeforeEach(func() {
			store = &Properties{m: make(map[string]string)}
			previous := commandlineProperties
			commandlineProperties = nil
			DeferCleanup(func() { commandlineProperties = previous })
		})

		It("honours a suffixed override that lowers the default cap", func() {
			store.Set("http.har.response.body.length", "1MiB")

			Expect(store.Bytes(4*1024*1024, "http.har.response.body.length")).To(Equal(1024 * 1024))
		})

		It("falls back to the default when the value does not parse", func() {
			store.Set("http.har.response.body.length", "not-a-size")

			Expect(store.Bytes(4*1024*1024, "http.har.response.body.length")).To(Equal(4 * 1024 * 1024))
		})

		It("returns an explicit 0 so truncation can be disabled", func() {
			store.Set("http.har.response.body.length", "0")

			Expect(store.Bytes(4*1024*1024, "http.har.response.body.length")).To(BeZero())
		})
	})
})
