package money

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type textAmount string

func (a textAmount) String() string { return string(a) }

func (a textAmount) Compare(other textAmount) int { return strings.Compare(string(a), string(other)) }

func parseTextAmount(raw string) (textAmount, error) { return textAmount(raw), nil }

var _ = Describe("Value.Combine", func() {
	newValue := func(amount string, places int) Value[textAmount] {
		value, err := New(Options{Amount: amount, Currency: "USD", DecimalPlaces: places}, parseTextAmount)
		Expect(err).ToNot(HaveOccurred())
		return value
	}

	It("keeps the wider display places of a successful result", func() {
		result, err := newValue("1", 2).Combine(newValue("2", 4), func(textAmount, textAmount) textAmount { return "3" })

		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(Value[textAmount]{amount: "3", currency: "USD", places: 4}))
	})

	It("returns the zero Value when the result is not finite", func() {
		result, err := newValue("1", 2).Combine(newValue("2", 4), func(textAmount, textAmount) textAmount { return "NaN" })

		Expect(err).To(MatchError(ContainSubstring("non-finite result NaN")))
		Expect(result).To(BeZero())
	})
})
