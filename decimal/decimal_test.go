package decimal_test

import (
	"encoding/json"
	"fmt"
	"github.com/flanksource/commons/bigdecimal"
	"github.com/flanksource/commons/decimal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDecimals(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "Decimal extensions") }

var _ = Describe("Decimal extensions", func() {
	It("retains Commons results and consistent formatting after arithmetic", func() {
		small, err := decimal.MustParse("1.20").Add(decimal.MustParse("0.03"))
		Expect(err).NotTo(HaveOccurred())
		large := bigdecimal.MustParse("1.20").Add(bigdecimal.MustParse("0.03"))
		for _, value := range []any{small, large} {
			Expect(fmt.Sprint(value)).To(Equal("1.23"))
			Expect(fmt.Sprintf("%.3f", value)).To(Equal("1.230"))
			data, err := json.Marshal(value)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(`"1.23"`))
		}
	})
	It("decodes exact numeric tokens atomically", func() {
		small := decimal.MustParse("7")
		large := bigdecimal.MustParse("7")
		for _, target := range []any{&small, &large} {
			Expect(json.Unmarshal([]byte(`9007199254740993.01`), target)).To(Succeed())
			Expect(fmt.Sprint(target)).To(Equal("9007199254740993.01"))
			for _, raw := range []string{`null`, `true`, `"NaN"`, `"bad"`} {
				Expect(json.Unmarshal([]byte(raw), target)).NotTo(Succeed())
				Expect(fmt.Sprint(target)).To(Equal("9007199254740993.01"))
			}
		}
	})
	It("uses backend precision and rejects overflow", func() {
		Expect(decimal.MustParse("9007199254740993.0000000001").String()).To(Equal("9007199254740993"))
		Expect(bigdecimal.MustParse("9007199254740993.0000000001").String()).To(Equal("9007199254740993.0000000001"))
		_, err := decimal.Parse("10000000000000000000")
		Expect(err).To(HaveOccurred())
		Expect(decimal.MustParse("0.00000000000000000005").String()).To(Equal("0"))
		_, err = decimal.MustParse("9999999999999999999").Add(decimal.MustParse("1"))
		Expect(err).To(HaveOccurred())
	})
	It("preserves formatting on multiple results and rejects nonfinite serialization", func() {
		q, r, err := decimal.MustParse("5").QuoRem(decimal.MustParse("2"))
		Expect(err).NotTo(HaveOccurred())
		Expect([]decimal.Decimal{q, r}).To(Equal([]decimal.Decimal{decimal.MustParse("2"), decimal.MustParse("1")}))
		bq, br := bigdecimal.MustParse("5").QuoRem(bigdecimal.MustParse("2"))
		Expect([]bigdecimal.Decimal{bq, br}).To(Equal([]bigdecimal.Decimal{bigdecimal.MustParse("2"), bigdecimal.MustParse("1")}))
		for _, value := range []any{decimal.MustParse("-1.20"), bigdecimal.MustParse("-1.20")} {
			Expect(fmt.Sprintf("%08.2f", value)).To(Equal("-0001.20"))
			Expect(fmt.Sprintf("%-7s", value)).To(Equal("-1.2   "))
		}
		for _, value := range []bigdecimal.Decimal{bigdecimal.NaN(), bigdecimal.Inf(1)} {
			_, err := json.Marshal(value)
			Expect(err).To(HaveOccurred())
			_, err = value.MarshalText()
			Expect(err).To(HaveOccurred())
		}
	})
	It("keeps both Money types currency aware with minimum display places", func() {
		small, err := decimal.NewMoney(decimal.MoneyOptions{Amount: "125.005", Currency: "USD", DecimalPlaces: 2})
		Expect(err).NotTo(HaveOccurred())
		large, err := bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: "125.005", Currency: "USD", DecimalPlaces: 2})
		Expect(err).NotTo(HaveOccurred())
		for _, value := range []any{small, large} {
			Expect(fmt.Sprint(value)).To(Equal("USD 125.005"))
			data, err := json.Marshal(value)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(`"USD 125.005"`))
		}
		other, err := decimal.NewMoney(decimal.MoneyOptions{Amount: "125.005", Currency: "EUR"})
		Expect(err).NotTo(HaveOccurred())
		_, err = small.Cmp(other)
		Expect(err).To(HaveOccurred())
		_, err = small.Cmp(decimal.Money{})
		Expect(err).To(HaveOccurred())
	})

	newMoney := map[string]func(amount, currency string, places int) (fmt.Stringer, error){
		"decimal": func(amount, currency string, places int) (fmt.Stringer, error) {
			return decimal.NewMoney(decimal.MoneyOptions{Amount: amount, Currency: currency, DecimalPlaces: places})
		},
		"bigdecimal": func(amount, currency string, places int) (fmt.Stringer, error) {
			return bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: amount, Currency: currency, DecimalPlaces: places})
		},
	}
	DescribeTable("rejects invalid Money options", func(amount, currency string, places int, message string) {
		for backend, construct := range newMoney {
			_, err := construct(amount, currency, places)
			Expect(err).To(MatchError(ContainSubstring(message)), backend)
		}
	},
		Entry("lowercase currency", "1.00", "usd", 2, "three-letter uppercase"),
		Entry("two-letter currency", "1.00", "US", 2, "three-letter uppercase"),
		Entry("four-letter currency", "1.00", "USDX", 2, "three-letter uppercase"),
		Entry("negative decimal places", "1.00", "USD", -1, "between 0 and 31"),
		Entry("decimal places above 31", "1.00", "USD", 32, "between 0 and 31"),
		Entry("exponent amount", "1e3", "USD", 2, "finite decimal"),
		Entry("NaN amount", "NaN", "USD", 2, "finite decimal"),
		Entry("Inf amount", "Inf", "USD", 2, "finite decimal"),
	)
	DescribeTable("orders same-currency Money", func(left, right string, expected int) {
		smallLeft, err := decimal.NewMoney(decimal.MoneyOptions{Amount: left, Currency: "USD"})
		Expect(err).NotTo(HaveOccurred())
		smallRight, err := decimal.NewMoney(decimal.MoneyOptions{Amount: right, Currency: "USD"})
		Expect(err).NotTo(HaveOccurred())
		largeLeft, err := bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: left, Currency: "USD"})
		Expect(err).NotTo(HaveOccurred())
		largeRight, err := bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: right, Currency: "USD"})
		Expect(err).NotTo(HaveOccurred())
		smallOrder, err := smallLeft.Cmp(smallRight)
		Expect(err).NotTo(HaveOccurred())
		largeOrder, err := largeLeft.Cmp(largeRight)
		Expect(err).NotTo(HaveOccurred())
		Expect([]int{smallOrder, largeOrder}).To(Equal([]int{expected, expected}))
	},
		Entry("less", "1.99", "2.00", -1),
		Entry("equal with different scale", "2.0", "2.00", 0),
		Entry("greater", "-1.00", "-1.01", 1),
	)
	It("formats zero-value Money as a visible uninitialized marker without panicking", func() {
		const marker = "<uninitialized Money>"
		Expect([]string{decimal.Money{}.String(), bigdecimal.Money{}.String()}).To(Equal([]string{marker, marker}))
		_, err := decimal.Money{}.MarshalJSON()
		Expect(err).To(MatchError(ContainSubstring("uninitialized Money")))
	})
})
