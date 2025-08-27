package collections

import (
	"net/url"
	"slices"
	"strings"
)

// Dedup removes duplicate elements from a slice while preserving order.
// Returns a new slice containing only the first occurrence of each element.
//
// Example:
//
//	nums := []int{1, 2, 2, 3, 1, 4}
//	unique := collections.Dedup(nums) // [1, 2, 3, 4]
func Dedup[T comparable](arr []T) []T {
	set := make(map[T]bool)
	retArr := []T{}
	for _, item := range arr {
		if _, value := set[item]; !value {
			set[item] = true
			retArr = append(retArr, item)
		}
	}
	return retArr
}

// ReplaceAllInSlice applies strings.ReplaceAll to each element in the slice.
//
// Example:
//
//	urls := []string{"http://api.com", "http://web.com"}
//	secure := collections.ReplaceAllInSlice(urls, "http://", "https://")
//	// Result: ["https://api.com", "https://web.com"]
func ReplaceAllInSlice(a []string, find string, replacement string) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.ReplaceAll(s, find, replacement))
	}
	return
}

// SplitAllInSlice splits each element and returns the part at the specified index.
//
// Example:
//
//	emails := []string{"john@example.com", "jane@test.org"}
//	domains := collections.SplitAllInSlice(emails, "@", 1)
//	// Result: ["example.com", "test.org"]
func SplitAllInSlice(a []string, split string, index int) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.Split(s, split)[index])
	}
	return
}

// Find returns the index of the first occurrence of x in the slice,
// or len(a) if x is not found.
//
// Example:
//
//	fruits := []string{"apple", "banana", "orange"}
//	idx := collections.Find(fruits, "banana") // Returns 1
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// Contains checks if a slice contains the specified element.
//
// Example:
//
//	nums := []int{1, 2, 3, 4, 5}
//	if collections.Contains(nums, 3) {
//		// Element found
//	}
func Contains[T comparable](a []T, x T) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// Append concatenates multiple slices of strings into a single slice.
func Append[T any](slices ...[]T) []T {
	if len(slices) == 0 {
		return nil
	}

	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}

	output := make([]T, 0, totalLen)
	for _, s := range slices {
		output = append(output, s...)
	}

	return output
}

func MatchAny(items []string, patterns ...string) (matches, negated bool) {
	var matched = false
	for _, item := range items {
		matches, negated = MatchItem(item, patterns...)
		matched = matched || matches
		if negated {
			return false, true
		}
	}

	return matched, false
}

// matchItems returns true if any of the patterns in the list match the item.
// negative matches are supported by prefixing the item with a "!" and
// takes precendence over positive match.
// * matches everything
// to match prefix and suffix use "*" accordingly.
func MatchItem(item string, patterns ...string) (matches, negated bool) {
	if len(patterns) == 0 {
		return true, false
	}

	slices.SortFunc(patterns, sortPatterns)

	//process negations first
	for _, p := range patterns {
		pattern, err := url.QueryUnescape(strings.TrimSpace(p))
		if err != nil {
			continue
		}

		if strings.HasPrefix(pattern, "!") {
			if matchPattern(item, strings.TrimPrefix(pattern, "!")) {
				return false, true
			}
		}

	}

	// then normal filters
	for _, p := range patterns {
		pattern, err := url.QueryUnescape(strings.TrimSpace(p))
		if err != nil {
			continue
		}

		if matchPattern(item, pattern) {
			return true, false
		}
	}

	//nolint:gosimple
	//lint:ignore S1008 ...
	if IsExclusionOnlyPatterns(patterns) {
		// If all the filters were exlusions, and none of the exclusions excluded the item, then it's a match
		return true, false
	}

	return false, false
}

// matchItems returns true if any of the patterns in the list match the item.
// negative matches are supported by prefixing the item with a "!" and
// takes precendence over positive match.
// * matches everything
// to match prefix and suffix use "*" accordingly.
func MatchItems(item string, patterns ...string) bool {
	if len(patterns) == 0 {
		return true
	}

	slices.SortFunc(patterns, sortPatterns)

	for _, p := range patterns {
		pattern, err := url.QueryUnescape(strings.TrimSpace(p))
		if err != nil {
			continue
		}

		if strings.HasPrefix(pattern, "!") {
			if matchPattern(item, strings.TrimPrefix(pattern, "!")) {
				return false
			}

			continue
		}

		if matchPattern(item, pattern) {
			return true
		}
	}

	//nolint:gosimple
	//lint:ignore S1008 ...
	if IsExclusionOnlyPatterns(patterns) {
		// If all the filters were exlusions, and none of the exclusions excluded the item, then it's a match
		return true
	}

	return false
}

func matchPattern(item, pattern string) bool {
	if pattern == "*" || item == pattern {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		if strings.Contains(item, strings.TrimPrefix(strings.TrimSuffix(pattern, "*"), "*")) {
			return true
		}
	}

	if strings.HasPrefix(pattern, "*") {
		if strings.HasSuffix(item, strings.TrimPrefix(pattern, "*")) {
			return true
		}
	}

	if strings.HasSuffix(pattern, "*") {
		if strings.HasPrefix(item, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}

	return false
}

// sortPatterns defines the priority for sorting:
// exclusions ("!") have higher priority than other patterns.
func sortPatterns(a, b string) int {
	if a == "!*" {
		return -1
	} else if b == "!*" {
		return 1
	}

	if strings.HasPrefix(a, "!") {
		return -1
	} else if strings.HasPrefix(b, "!") {
		return 1
	}

	return 0
}

func IsExclusionOnlyPatterns(patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return false
		}

		if !strings.HasPrefix(pattern, "!") {
			return false
		}
	}

	return true
}

func DeleteEmptyStrings(s []string) []string {
	r := make([]string, 0, len(s))
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}

	return r
}
