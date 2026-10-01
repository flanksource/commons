/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Ported from k8s.io/apimachinery v0.36.2 pkg/util/strategicpatch/patch.go; apply half only.

package strategicpatch

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// validateMergeKeyInLists checks if each map in the list has the mentryerge key.
func validateMergeKeyInLists(mergeKey string, lists ...[]interface{}) error {
	for _, list := range lists {
		for _, item := range list {
			m, ok := item.(map[string]interface{})
			if !ok {
				return mergepatch.ErrBadArgType(m, item)
			}
			if _, ok = m[mergeKey]; !ok {
				return mergepatch.ErrNoMergeKey(m, mergeKey)
			}
		}
	}
	return nil
}

// normalizeElementOrder sort `patch` list by `patchOrder` and sort `serverOnly` list by `serverOrder`.
// Then it merges the 2 sorted lists.
// It guarantee the relative order in the patch list and in the serverOnly list is kept.
// `patch` is a list of items in the patch, and `serverOnly` is a list of items in the live object.
// `patchOrder` is the order we want `patch` list to have and
// `serverOrder` is the order we want `serverOnly` list to have.
// kind is the kind of each item in the lists `patch` and `serverOnly`.
func normalizeElementOrder(patch, serverOnly, patchOrder, serverOrder []interface{}, mergeKey string, kind reflect.Kind) ([]interface{}, error) {
	patch, err := normalizeSliceOrder(patch, patchOrder, mergeKey, kind)
	if err != nil {
		return nil, err
	}
	serverOnly, err = normalizeSliceOrder(serverOnly, serverOrder, mergeKey, kind)
	if err != nil {
		return nil, err
	}
	all := mergeSortedSlice(serverOnly, patch, serverOrder, mergeKey, kind)

	return all, nil
}

// mergeSortedSlice merges the 2 sorted lists by serverOrder with best effort.
// It will insert each item in `left` list to `right` list. In most cases, the 2 lists will be interleaved.
// The relative order of left and right are guaranteed to be kept.
// They have higher precedence than the order in the live list.
// The place for a item in `left` is found by:
// scan from the place of last insertion in `right` to the end of `right`,
// the place is before the first item that is greater than the item we want to insert.
// example usage: using server-only items as left and patch items as right. We insert server-only items
// to patch list. We use the order of live object as record for comparison.
func mergeSortedSlice(left, right, serverOrder []interface{}, mergeKey string, kind reflect.Kind) []interface{} {
	// Returns if l is less than r, and if both have been found.
	// If l and r both present and l is in front of r, l is less than r.
	less := func(l, r interface{}) (bool, bool) {
		li := index(serverOrder, l, mergeKey, kind)
		ri := index(serverOrder, r, mergeKey, kind)
		if li >= 0 && ri >= 0 {
			return li < ri, true
		} else {
			return false, false
		}
	}

	// left and right should be non-overlapping.
	size := len(left) + len(right)
	i, j := 0, 0
	s := make([]interface{}, size)

	for k := 0; k < size; k++ {
		if i >= len(left) && j < len(right) {
			// have items left in `right` list
			s[k] = right[j]
			j++
		} else if j >= len(right) && i < len(left) {
			// have items left in `left` list
			s[k] = left[i]
			i++
		} else {
			// compare them if i and j are both in bound
			less, foundBoth := less(left[i], right[j])
			if foundBoth && less {
				s[k] = left[i]
				i++
			} else {
				s[k] = right[j]
				j++
			}
		}
	}
	return s
}

// index returns the index of the item in the given items, or -1 if it doesn't exist
// l must NOT be a slice of slices, this should be checked before calling.
func index(l []interface{}, valToLookUp interface{}, mergeKey string, kind reflect.Kind) int {
	var getValFn func(interface{}) interface{}
	// Get the correct `getValFn` based on item `kind`.
	// It should return the value of merge key for maps and
	// return the item for other kinds.
	switch kind {
	case reflect.Map:
		getValFn = func(item interface{}) interface{} {
			typedItem, ok := item.(map[string]interface{})
			if !ok {
				return nil
			}
			val := typedItem[mergeKey]
			return val
		}
	default:
		getValFn = func(item interface{}) interface{} {
			return item
		}
	}

	for i, v := range l {
		if getValFn(valToLookUp) == getValFn(v) {
			return i
		}
	}
	return -1
}

// extractToDeleteItems takes a list and
// returns 2 lists: one contains items that should be kept and the other contains items to be deleted.
func extractToDeleteItems(l []interface{}) ([]interface{}, []interface{}, error) {
	var nonDelete, toDelete []interface{}
	for _, v := range l {
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, nil, mergepatch.ErrBadArgType(m, v)
		}

		directive, foundDirective := m[directiveMarker]
		if foundDirective && directive == deleteDirective {
			toDelete = append(toDelete, v)
		} else {
			nonDelete = append(nonDelete, v)
		}
	}
	return nonDelete, toDelete, nil
}

// normalizeSliceOrder sort `toSort` list by `order`
func normalizeSliceOrder(toSort, order []interface{}, mergeKey string, kind reflect.Kind) ([]interface{}, error) {
	var toDelete []interface{}
	if kind == reflect.Map {
		// make sure each item in toSort, order has merge key
		err := validateMergeKeyInLists(mergeKey, toSort, order)
		if err != nil {
			return nil, err
		}
		toSort, toDelete, err = extractToDeleteItems(toSort)
		if err != nil {
			return nil, err
		}
	}

	sort.SliceStable(toSort, func(i, j int) bool {
		if ii := index(order, toSort[i], mergeKey, kind); ii >= 0 {
			if ij := index(order, toSort[j], mergeKey, kind); ij >= 0 {
				return ii < ij
			}
		}
		return true
	})
	toSort = append(toSort, toDelete...)
	return toSort, nil
}

func deduplicateScalars(s []interface{}) []interface{} {
	// Clever algorithm to deduplicate.
	length := len(s) - 1
	for i := 0; i < length; i++ {
		for j := i + 1; j <= length; j++ {
			if s[i] == s[j] {
				s[j] = s[length]
				s = s[0:length]
				length--
				j--
			}
		}
	}

	return s
}

// Returns the type of the elements of N slice(s). If the type is different,
// another slice or undefined, returns an error.
func sliceElementType(slices ...[]interface{}) (reflect.Type, error) {
	var prevType reflect.Type
	for _, s := range slices {
		// Go through elements of all given slices and make sure they are all the same type.
		for _, v := range s {
			currentType := reflect.TypeOf(v)
			if prevType == nil {
				prevType = currentType
				// We don't support lists of lists yet.
				if prevType.Kind() == reflect.Slice {
					return nil, mergepatch.ErrNoListOfLists
				}
			} else {
				if prevType != currentType {
					return nil, fmt.Errorf("list element types are not identical: %v", fmt.Sprint(slices))
				}
				prevType = currentType
			}
		}
	}

	if prevType == nil {
		return nil, fmt.Errorf("no elements in any of the given slices")
	}

	return prevType, nil
}
