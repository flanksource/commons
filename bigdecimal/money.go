package bigdecimal

// Cmp compares amounts in the same currency and rejects uninitialized values.
func (m Money) Cmp(other Money) (int, error) { return m.Value.Cmp(other.Value) }

// Add sums amounts in the same currency; the result keeps the wider display places.
func (m Money) Add(other Money) (Money, error) { return m.combine(other, Decimal.Add) }

// Sub subtracts an amount in the same currency; the result keeps the wider display places.
func (m Money) Sub(other Money) (Money, error) { return m.combine(other, Decimal.Sub) }

// Neg returns the amount with its sign flipped.
func (m Money) Neg() (Money, error) { return m.scale(m.Amount().Neg()) }

// Mul multiplies the amount by a plain factor, keeping the currency.
func (m Money) Mul(factor Decimal) (Money, error) { return m.scale(m.Amount().Mul(factor)) }

// Quo divides the amount by a plain divisor; a zero divisor is an error.
func (m Money) Quo(divisor Decimal) (Money, error) { return m.scale(m.Amount().Quo(divisor)) }

func (m Money) combine(other Money, op func(Decimal, Decimal) Decimal) (Money, error) {
	value, err := m.Value.Combine(other.Value, op)
	return Money{Value: value}, err
}

func (m Money) scale(amount Decimal) (Money, error) {
	value, err := m.WithAmount(amount)
	return Money{Value: value}, err
}
