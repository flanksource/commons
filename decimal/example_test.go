package decimal_test

import (
	"fmt"
	"github.com/flanksource/commons/bigdecimal"
	"github.com/flanksource/commons/decimal"
)

func ExampleNewMoney() {
	premium, err := decimal.NewMoney(decimal.MoneyOptions{Amount: "125", Currency: "USD", DecimalPlaces: 2})
	if err != nil {
		panic(err)
	}
	fmt.Println(premium)
	fmt.Println(premium.Amount())
	// Output:
	// USD 125.00
	// 125
}
func ExampleDecimal_Add() {
	total, err := decimal.MustParse("1.20").Add(decimal.MustParse("0.03"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s %.3f\n", total, total)
	fmt.Println(bigdecimal.MustParse("9007199254740993.0000000001"))
	// Output:
	// 1.23 1.230
	// 9007199254740993.0000000001
}
