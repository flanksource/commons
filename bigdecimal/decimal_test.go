package bigdecimal_test

import (
	"github.com/flanksource/commons/bigdecimal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("Decimal.Compare orders every value without panicking", func(d, o bigdecimal.Decimal, expected int) {
	Expect(d.Compare(o)).To(Equal(expected))
	Expect(bigdecimal.Compare(d, o)).To(Equal(expected))
},
	Entry("less", bigdecimal.MustParse("1.99"), bigdecimal.MustParse("2"), -1),
	Entry("equal with different scale", bigdecimal.MustParse("2.0"), bigdecimal.MustParse("2.00"), 0),
	Entry("greater", bigdecimal.MustParse("-1"), bigdecimal.MustParse("-1.01"), 1),
	Entry("NaN below a number", bigdecimal.NaN(), bigdecimal.MustParse("-1"), -1),
	Entry("number above NaN", bigdecimal.MustParse("-1"), bigdecimal.NaN(), 1),
	Entry("NaN below negative infinity", bigdecimal.NaN(), bigdecimal.Inf(-1), -1),
	Entry("NaN equal to NaN", bigdecimal.NaN(), bigdecimal.NaN(), 0),
	Entry("positive infinity above a number", bigdecimal.Inf(1), bigdecimal.MustParse("1e100"), 1),
)
