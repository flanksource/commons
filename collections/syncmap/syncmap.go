package syncmap

import "sync"

// SyncMap is like a Go sync.Map but type-safe using generics.
//
// The zero SyncMap is empty and ready for use. A SyncMap must not be copied after first use.
type SyncMap[K comparable, V any] struct {
	m     sync.Map
	zeroV V
}

func New[K comparable, V any]() SyncMap[K, V] {
	return SyncMap[K, V]{}
}

func (s *SyncMap[K, V]) Clear() {
	s.m.Clear()
}

func (s *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := s.m.Load(key)
	if !ok {
		return s.zeroV, false
	}
	return v.(V), true

}

// Store sets the value for a key.
func (s *SyncMap[K, V]) Store(key K, value V) {
	s.m.Store(key, value)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (s *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := s.m.LoadOrStore(key, value)
	if !loaded {
		return value, false
	}
	return v.(V), true
}

// Delete deletes the value for a key.
func (s *SyncMap[K, V]) Delete(key K) {
	s.m.Delete(key)
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (s *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := s.m.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
// The old value must be of a comparable type.
//
// If there is no current value for key in the map, CompareAndDelete returns false
// (even if the old value is the nil interface value).
func (s *SyncMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return s.m.CompareAndDelete(key, old)
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (s *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, loaded := s.m.Swap(key, value)
	if !loaded {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
// The old value must be of a comparable type.
func (s *SyncMap[K, V]) CompareAndSwap(key K, old, new V) bool {
	return s.m.CompareAndSwap(key, old, new)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently (including by f), Range may reflect any
// mapping for that key from any point during the Range call. Range does not
// block other methods on the receiver; even f itself may call any method on m.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (s *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}
