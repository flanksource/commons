package collections

import (
	"net/url"
	"slices"
	"strings"
)

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

// ReplaceAllInSlice runs strings.Replace on all elements in a slice and returns the result
func ReplaceAllInSlice(a []string, find string, replacement string) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.ReplaceAll(s, find, replacement))
	}
	return
}

// SplitAllInSlice runs strings.Split on all elements in a slice and returns the results at the given index
func SplitAllInSlice(a []string, split string, index int) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.Split(s, split)[index])
	}
	return
}

// Find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// Contains tells whether a contains x.
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
		if strings.Contains(item, strings.Trim(pattern, "*")) {
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
