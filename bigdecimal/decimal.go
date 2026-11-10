// Package bigdecimal extends decimal128 with canonical presentation and Money.
// Native arithmetic and rounding semantics are preserved, including special values.
package bigdecimal

import (
	"encoding/json"
	"fmt"

	"github.com/flanksource/commons/internal/decimalformat"
	"github.com/flanksource/commons/internal/money"
	upstream "github.com/woodsbury/decimal128"
)

type Decimal struct{ upstream.Decimal }

var Zero = Decimal{}
var One = FromInt64(1)

func (d Decimal) String() string {
	if d.IsZero() {
		return "0"
	}
	return upstream.Format(d.Decimal, 'f', -1)
}

// Compare returns -1, 0 or 1 as d is less than, equal to or greater than
// other. It panics when either operand is NaN, which has no order; check
// IsNaN first, or use the package-level Compare for upstream NaN ordering.
func (d Decimal) Compare(other Decimal) int {
	r := d.Decimal.Cmp(other.Decimal)
	if r.Less() {
		return -1
	}
	if r.Greater() {
		return 1
	}
	if r.Equal() {
		return 0
	}
	panic("cannot compare NaN")
}
func (d Decimal) Format(state fmt.State, verb rune) {
	decimalformat.Presentation{Text: d.String(), Fixed: func(p int) string { return decimalformat.Fixed(d.Decimal, p) }}.Format(state, verb)
}
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.IsNaN() || d.IsInf(0) {
		return nil, fmt.Errorf("cannot serialize nonfinite decimal")
	}
	return json.Marshal(d.String())
}
func (d *Decimal) UnmarshalJSON(data []byte) error {
	text, err := decimalformat.Decode(data)
	if err != nil {
		return err
	}
	return d.UnmarshalText([]byte(text))
}
func (d Decimal) MarshalText() ([]byte, error) {
	if d.IsNaN() || d.IsInf(0) {
		return nil, fmt.Errorf("cannot serialize nonfinite decimal")
	}
	return []byte(d.String()), nil
}
func (d Decimal) AppendText(dst []byte) ([]byte, error) {
	text, err := d.MarshalText()
	if err != nil {
		return nil, err
	}
	return append(dst, text...), nil
}
func (d *Decimal) UnmarshalText(data []byte) error {
	value, err := Parse(string(data))
	if err != nil {
		return err
	}
	if value.IsNaN() || value.IsInf(0) {
		return fmt.Errorf("expected a finite decimal")
	}
	*d = value
	return nil
}

type MoneyOptions = money.Options
type Money struct{ money.Value[Decimal] }

func NewMoney(opts MoneyOptions) (Money, error) {
	value, err := money.New(opts, Parse)
	return Money{Value: value}, err
}
