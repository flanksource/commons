package collections

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SortedMap", func() {
	It("should sort map entries alphabetically", func() {
		labels := map[string]string{
			"b": "b",
			"a": "a",
			"c": "c",
		}

		result := SortedMap(labels)

		Expect(result).To(Equal("a=a,b=b,c=c"))
	})
})
