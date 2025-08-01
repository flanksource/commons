// Package collections provides generic utilities for working with
// collections like slices, maps, sets, and priority queues.
//
// The package leverages Go generics to provide type-safe operations
// without runtime overhead. It includes utilities for common operations
// like filtering, mapping, grouping, and set operations.
//
// Slice Operations:
//
//	numbers := []int{1, 2, 3, 4, 5}
//	evens := collections.Filter(numbers, func(n int) bool { return n%2 == 0 })
//	squares := collections.Map(numbers, func(n int) int { return n * n })
//
// Map Operations:
//
//	data := map[string]int{"a": 1, "b": 2}
//	keys := collections.Keys(data)
//	values := collections.Values(data)
//
// Set Operations:
//
//	set1 := collections.NewSet(1, 2, 3)
//	set2 := collections.NewSet(2, 3, 4)
//	union := set1.Union(set2)      // {1, 2, 3, 4}
//	intersect := set1.Intersect(set2) // {2, 3}
package collections

import (
	"fmt"
	"reflect"
)

func takeSliceArg(arg interface{}) (out []interface{}, ok bool) {
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Slice {
		return nil, false
	}

	c := val.Len()
	out = make([]interface{}, c)
	for i := 0; i < val.Len(); i++ {
		out[i] = val.Index(i).Interface()
	}
	return out, true
}

// ToString converts various types to their string representation.
// Handles slices by joining elements with commas, fmt.Stringer types,
// strings, booleans, and falls back to fmt.Sprintf for other types.
//
// Example:
//
//	collections.ToString([]int{1, 2, 3})     // "1, 2, 3"
//	collections.ToString(true)              // "true"
//	collections.ToString(nil)               // ""
func ToString(i interface{}) string {
	if slice, ok := takeSliceArg(i); ok {
		s := ""
		for _, v := range slice {
			if s != "" {
				s += ", "
			}
			s += ToString(v)
		}
		return s

	}
	switch v := i.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case interface{}:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	default:
		// panic(fmt.Sprintf("I don't know about type %T!\n", v))
	}
	return ""
}
