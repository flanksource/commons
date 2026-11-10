# Bigdecimal

`github.com/flanksource/commons/bigdecimal` extends `github.com/woodsbury/decimal128` with Commons arithmetic results, canonical formatting, quoted JSON, and currency-aware Money. See [Decimal and Money](../decimal/README.md) for the shared API, examples, and serialization rules.

`bigdecimal.Money` also does arithmetic. `Add` and `Sub` take another Money in the same currency and keep whichever operand shows more decimal places. `Mul`, `Quo` and `Neg` take a plain `Decimal`, and the result stays in the Money's currency. Mixing currencies, using an uninitialized Money, or getting a non-finite result (such as dividing by zero) returns an error.

```go
fee, _ := bigdecimal.NewMoney(bigdecimal.MoneyOptions{Amount: "12.5", Currency: "ZAR", DecimalPlaces: 2})
total, err := fee.Mul(bigdecimal.MustParse("3")) // ZAR 37.50
```
