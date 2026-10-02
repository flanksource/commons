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
// These helpers only order merge lists so tests can compare results; the apply path never calls them.

package strategicpatch

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/flanksource/commons/merge/strategicpatch/internal/json"
	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// This function takes a JSON map and sorts all the lists that should be merged
// by key. This is needed by tests because in JSON, list order is significant,
// but in Strategic Merge Patch, merge lists do not have significant order.
// Sorting the lists allows for order-insensitive comparison of patched maps.
func sortMergeListsByName(mapJSON []byte, schema LookupPatchMeta) ([]byte, error) {
	var m map[string]interface{}
	err := json.Unmarshal(mapJSON, &m)
	if err != nil {
		return nil, mergepatch.ErrBadJSONDoc
	}

	newM, err := sortMergeListsByNameMap(m, schema)
	if err != nil {
		return nil, err
	}

	return json.Marshal(newM)
}

// Function sortMergeListsByNameMap recursively sorts the merge lists by its mergeKey in a map.
func sortMergeListsByNameMap(s map[string]interface{}, schema LookupPatchMeta) (map[string]interface{}, error) {
	newS := map[string]interface{}{}
	for k, v := range s {
		if k == retainKeysDirective {
			typedV, ok := v.([]interface{})
			if !ok {
				return nil, mergepatch.ErrBadPatchFormatForRetainKeys
			}
			v = sortScalars(typedV)
		} else if strings.HasPrefix(k, deleteFromPrimitiveListDirectivePrefix) {
			typedV, ok := v.([]interface{})
			if !ok {
				return nil, mergepatch.ErrBadPatchFormatForPrimitiveList
			}
			v = sortScalars(typedV)
		} else if strings.HasPrefix(k, setElementOrderDirectivePrefix) {
			_, ok := v.([]interface{})
			if !ok {
				return nil, mergepatch.ErrBadPatchFormatForSetElementOrderList
			}
		} else if k != directiveMarker {
			// recurse for map and slice.
			switch typedV := v.(type) {
			case map[string]interface{}:
				subschema, _, err := schema.LookupPatchMetadataForStruct(k)
				if err != nil {
					return nil, err
				}
				v, err = sortMergeListsByNameMap(typedV, subschema)
				if err != nil {
					return nil, err
				}
			case []interface{}:
				subschema, patchMeta, err := schema.LookupPatchMetadataForSlice(k)
				if err != nil {
					return nil, err
				}
				_, patchStrategy, err := extractRetainKeysPatchStrategy(patchMeta.GetPatchStrategies())
				if err != nil {
					return nil, err
				}
				if patchStrategy == mergeDirective {
					var err error
					v, err = sortMergeListsByNameArray(typedV, subschema, patchMeta.GetPatchMergeKey(), true)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		newS[k] = v
	}

	return newS, nil
}

// Function sortMergeListsByNameMap recursively sorts the merge lists by its mergeKey in an array.
func sortMergeListsByNameArray(s []interface{}, schema LookupPatchMeta, mergeKey string, recurse bool) ([]interface{}, error) {
	if len(s) == 0 {
		return s, nil
	}

	// We don't support lists of lists yet.
	t, err := sliceElementType(s)
	if err != nil {
		return nil, err
	}

	// If the elements are not maps...
	if t.Kind() != reflect.Map {
		// Sort the elements, because they may have been merged out of order.
		return deduplicateAndSortScalars(s), nil
	}

	// Elements are maps - if one of the keys of the map is a map or a
	// list, we may need to recurse into it.
	newS := []interface{}{}
	for _, elem := range s {
		if recurse {
			typedElem := elem.(map[string]interface{})
			newElem, err := sortMergeListsByNameMap(typedElem, schema)
			if err != nil {
				return nil, err
			}

			newS = append(newS, newElem)
		} else {
			newS = append(newS, elem)
		}
	}

	// Sort the maps.
	newS = sortMapsBasedOnField(newS, mergeKey)
	return newS, nil
}

func sortMapsBasedOnField(m []interface{}, fieldName string) []interface{} {
	mapM := mapSliceFromSlice(m)
	ss := SortableSliceOfMaps{mapM, fieldName}
	sort.Sort(ss)
	newS := sliceFromMapSlice(ss.s)
	return newS
}

func mapSliceFromSlice(m []interface{}) []map[string]interface{} {
	newM := []map[string]interface{}{}
	for _, v := range m {
		vt := v.(map[string]interface{})
		newM = append(newM, vt)
	}

	return newM
}

func sliceFromMapSlice(s []map[string]interface{}) []interface{} {
	newS := []interface{}{}
	for _, v := range s {
		newS = append(newS, v)
	}

	return newS
}

type SortableSliceOfMaps struct {
	s []map[string]interface{}
	k string // key to sort on
}

func (ss SortableSliceOfMaps) Len() int {
	return len(ss.s)
}

func (ss SortableSliceOfMaps) Less(i, j int) bool {
	iStr := fmt.Sprintf("%v", ss.s[i][ss.k])
	jStr := fmt.Sprintf("%v", ss.s[j][ss.k])
	return sort.StringsAreSorted([]string{iStr, jStr})
}

func (ss SortableSliceOfMaps) Swap(i, j int) {
	ss.s[i], ss.s[j] = ss.s[j], ss.s[i]
}

func deduplicateAndSortScalars(s []interface{}) []interface{} {
	s = deduplicateScalars(s)
	return sortScalars(s)
}

func sortScalars(s []interface{}) []interface{} {
	ss := SortableSliceOfScalars{s}
	sort.Sort(ss)
	return ss.s
}

type SortableSliceOfScalars struct {
	s []interface{}
}

func (ss SortableSliceOfScalars) Len() int {
	return len(ss.s)
}

func (ss SortableSliceOfScalars) Less(i, j int) bool {
	iStr := fmt.Sprintf("%v", ss.s[i])
	jStr := fmt.Sprintf("%v", ss.s[j])
	return sort.StringsAreSorted([]string{iStr, jStr})
}

func (ss SortableSliceOfScalars) Swap(i, j int) {
	ss.s[i], ss.s[j] = ss.s[j], ss.s[i]
}
