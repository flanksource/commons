package collections

import (
	"net/url"
	"strings"

	"github.com/flanksource/commons/logger"
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
		replaced = append(replaced, strings.Replace(s, find, replacement, -1))
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

// matchItems returns true if any of the items in the list match the item.
// negative matches are supported by prefixing the item with a "!".
// * matches everything
// to match prefix and suffix use "*" accordingly.
func MatchItems(item string, items ...string) bool {
	if len(items) == 0 {
		return true
	}

	for _, i := range items {
		i = strings.TrimSpace(i)

		i, err := url.QueryUnescape(i)
		if err != nil {
			logger.Warnf("match items received item with invalid url encoding: %v", err)
			continue
		}

		if strings.HasPrefix(i, "!") {
			if item == strings.TrimPrefix(i, "!") {
				return false
			}

			continue
		}

		if i == "*" || item == i {
			return true
		}

		if strings.HasPrefix(i, "*") {
			if strings.HasSuffix(item, strings.TrimPrefix(i, "*")) {
				return true
			}
		}

		if strings.HasSuffix(i, "*") {
			if strings.HasPrefix(item, strings.TrimSuffix(i, "*")) {
				return true
			}
		}
	}

	return false
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
