// Package decimal extends govalues decimals with canonical presentation and Money.
// Values use 19-digit half-even rounding; arithmetic retains Commons result types.
package decimal

import (
	"encoding/json"
	"fmt"

	"github.com/flanksource/commons/internal/decimalformat"
	"github.com/flanksource/commons/internal/money"
	upstream "github.com/govalues/decimal"
)

type Decimal struct{ upstream.Decimal }

var Zero = Decimal{upstream.Zero}
var One = Decimal{upstream.One}

const MaxPrec = upstream.MaxPrec
const MaxScale = upstream.MaxScale
const MinScale = upstream.MinScale

func (d Decimal) String() string            { return d.Decimal.Trim(0).String() }
func (d Decimal) Compare(other Decimal) int { return d.Decimal.Cmp(other.Decimal) }
func (d Decimal) Format(state fmt.State, verb rune) {
	decimalformat.Presentation{Text: d.String(), Fixed: func(p int) string { return decimalformat.Fixed(d.Decimal, p) }}.Format(state, verb)
}
func (d Decimal) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }
func (d *Decimal) UnmarshalJSON(data []byte) error {
	text, err := decimalformat.Decode(data)
	if err != nil {
		return err
	}
	return d.UnmarshalText([]byte(text))
}
func (d Decimal) MarshalText() ([]byte, error)          { return []byte(d.String()), nil }
func (d Decimal) AppendText(dst []byte) ([]byte, error) { return append(dst, d.String()...), nil }
func (d *Decimal) UnmarshalText(data []byte) error {
	value, err := Parse(string(data))
	if err != nil {
		return err
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
