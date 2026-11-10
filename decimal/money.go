package decimal

// Cmp compares amounts in the same currency and rejects uninitialized values.
func (m Money) Cmp(other Money) (int, error) { return m.Value.Cmp(other.Value) }
