package money

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var amountPattern = regexp.MustCompile(`^[+-]?[0-9]+(?:\.[0-9]+)?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type Options struct {
	Amount        string
	Currency      string
	DecimalPlaces int
}
type Amount[T any] interface {
	String() string
	Compare(T) int
}
type Value[T Amount[T]] struct {
	amount   T
	currency string
	places   int
}

func New[T Amount[T]](opts Options, parse func(string) (T, error)) (Value[T], error) {
	if !amountPattern.MatchString(opts.Amount) {
		return Value[T]{}, fmt.Errorf("money amount must be a finite decimal, got %q", opts.Amount)
	}
	if !currencyPattern.MatchString(opts.Currency) {
		return Value[T]{}, fmt.Errorf("money currency must be a three-letter uppercase code, got %q", opts.Currency)
	}
	if opts.DecimalPlaces < 0 || opts.DecimalPlaces > 31 {
		return Value[T]{}, fmt.Errorf("money decimal places must be between 0 and 31, got %d", opts.DecimalPlaces)
	}
	amount, err := parse(opts.Amount)
	if err != nil {
		return Value[T]{}, fmt.Errorf("money amount: %w", err)
	}
	if !finite(amount) {
		return Value[T]{}, fmt.Errorf("money amount must be finite")
	}
	return Value[T]{amount: amount, currency: opts.Currency, places: opts.DecimalPlaces}, nil
}

func finite[T Amount[T]](amount T) bool {
	return amount.String() != "NaN" && !strings.Contains(amount.String(), "Inf")
}

// WithAmount returns amount in m's currency with at least m's display places.
func (m Value[T]) WithAmount(amount T) (Value[T], error) {
	if err := m.Validate(); err != nil {
		return Value[T]{}, err
	}
	if !finite(amount) {
		return Value[T]{}, fmt.Errorf("money arithmetic on %s produced a non-finite result %s", m, amount)
	}
	return Value[T]{amount: amount, currency: m.currency, places: m.places}, nil
}

// Combine applies op to two amounts of the same currency; the result keeps the wider display places.
func (m Value[T]) Combine(other Value[T], op func(T, T) T) (Value[T], error) {
	if err := sameCurrency("combine", m, other); err != nil {
		return Value[T]{}, err
	}
	result, err := m.WithAmount(op(m.amount, other.amount))
	result.places = max(m.places, other.places)
	return result, err
}

func sameCurrency[T Amount[T]](verb string, m, other Value[T]) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := other.Validate(); err != nil {
		return err
	}
	if m.currency != other.currency {
		return fmt.Errorf("cannot %s different currencies: %s and %s", verb, m.currency, other.currency)
	}
	return nil
}
func (m Value[T]) Amount() T          { return m.amount }
func (m Value[T]) Currency() string   { return m.currency }
func (m Value[T]) DecimalPlaces() int { return m.places }
func (m Value[T]) Validate() error {
	if m.currency == "" {
		return fmt.Errorf("uninitialized Money; use NewMoney")
	}
	return nil
}
func (m Value[T]) String() string {
	if err := m.Validate(); err != nil {
		panic(err)
	}
	text := m.amount.String()
	_, fraction, _ := strings.Cut(text, ".")
	if len(fraction) < m.places {
		if fraction == "" {
			text += "."
		}
		text += strings.Repeat("0", m.places-len(fraction))
	}
	return m.currency + " " + text
}
func (m Value[T]) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}
func (m Value[T]) MarshalText() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return []byte(m.String()), nil
}
func (m Value[T]) Cmp(other Value[T]) (int, error) {
	if err := sameCurrency("compare", m, other); err != nil {
		return 0, err
	}
	return m.amount.Compare(other.amount), nil
}
