package bigdecimal_test

import (
	"testing"

	"github.com/flanksource/commons/bigdecimal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBigdecimal(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "bigdecimal") }

func money(amount, currency string, places int) bigdecimal.Money {
	value, err := bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: amount, Currency: currency, DecimalPlaces: places})
	Expect(err).NotTo(HaveOccurred())
	return value
}

var _ = Describe("Money arithmetic", func() {
	It("adds and subtracts exactly, keeping the widest display places", func() {
		sum, err := money("125.005", "USD", 2).Add(money("0.005", "USD", 3))
		Expect(err).NotTo(HaveOccurred())
		Expect(sum.String()).To(Equal("USD 125.010"))
		difference, err := money("9007199254740993.0000000001", "USD", 2).Sub(money("1", "USD", 0))
		Expect(err).NotTo(HaveOccurred())
		Expect(difference.String()).To(Equal("USD 9007199254740992.0000000001"))
	})
	It("multiplies, divides and negates by a plain decimal in the same currency", func() {
		product, err := money("12.5", "ZAR", 2).Mul(bigdecimal.MustParse("3"))
		Expect(err).NotTo(HaveOccurred())
		Expect(product.String()).To(Equal("ZAR 37.50"))
		quotient, err := money("10", "ZAR", 2).Quo(bigdecimal.MustParse("4"))
		Expect(err).NotTo(HaveOccurred())
		Expect(quotient.String()).To(Equal("ZAR 2.50"))
		negated, err := money("2.5", "ZAR", 2).Neg()
		Expect(err).NotTo(HaveOccurred())
		Expect(negated.String()).To(Equal("ZAR -2.50"))
	})
	It("rejects different currencies, uninitialized values and non-finite results", func() {
		_, err := money("1", "USD", 2).Add(money("1", "EUR", 2))
		Expect(err).To(MatchError(ContainSubstring("different currencies: USD and EUR")))
		_, err = money("1", "USD", 2).Sub(bigdecimal.Money{})
		Expect(err).To(MatchError(ContainSubstring("uninitialized Money")))
		_, err = money("1", "USD", 2).Quo(bigdecimal.Zero)
		Expect(err).To(MatchError(ContainSubstring("finite")))
	})
})
