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

	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// mergeSliceHandler handles how to merge `patchV` whose key is `key` with `original` respecting
// fieldPatchStrategy, fieldPatchMergeKey, isDeleteList and mergeOptions.
func mergeSliceHandler(original, patch interface{}, schema LookupPatchMeta,
	fieldPatchStrategy, fieldPatchMergeKey string, isDeleteList bool, mergeOptions MergeOptions) ([]interface{}, error) {
	typedOriginal, typedPatch, err := sliceTypeAssertion(original, patch)
	if err != nil {
		return nil, err
	}

	// Delete lists are handled the same way regardless of what the field's patch strategy is
	if fieldPatchStrategy == mergeDirective || isDeleteList {
		return mergeSlice(typedOriginal, typedPatch, schema, fieldPatchMergeKey, mergeOptions, isDeleteList)
	} else {
		return typedPatch, nil
	}
}

// Merge two slices together. Note: This may modify both the original slice and
// the patch because getting a deep copy of a slice in golang is highly
// non-trivial.
func mergeSlice(original, patch []interface{}, schema LookupPatchMeta, mergeKey string, mergeOptions MergeOptions, isDeleteList bool) ([]interface{}, error) {
	if len(original) == 0 && len(patch) == 0 {
		return original, nil
	}

	// All the values must be of the same type, but not a list.
	t, err := sliceElementType(original, patch)
	if err != nil {
		return nil, err
	}

	var merged []interface{}
	kind := t.Kind()
	// If the elements are not maps, merge the slices of scalars.
	if kind != reflect.Map {
		if mergeOptions.MergeParallelList && isDeleteList {
			return deleteFromSlice(original, patch), nil
		}
		// Maybe in the future add a "concat" mode that doesn't
		// deduplicate.
		both := append(original, patch...)
		merged = deduplicateScalars(both)

	} else {
		if mergeKey == "" {
			return nil, fmt.Errorf("cannot merge lists without merge key for %s", schema.Name())
		}

		original, patch, err = mergeSliceWithSpecialElements(original, patch, mergeKey)
		if err != nil {
			return nil, err
		}

		merged, err = mergeSliceWithoutSpecialElements(original, patch, mergeKey, schema, mergeOptions)
		if err != nil {
			return nil, err
		}
	}

	// enforce the order
	var patchItems, serverOnlyItems []interface{}
	if len(mergeKey) == 0 {
		patchItems, serverOnlyItems = partitionPrimitivesByPresentInList(merged, patch)
	} else {
		patchItems, serverOnlyItems, err = partitionMapsByPresentInList(merged, patch, mergeKey)
		if err != nil {
			return nil, err
		}
	}
	return normalizeElementOrder(patchItems, serverOnlyItems, patch, original, mergeKey, kind)
}

// mergeSliceWithSpecialElements handles special elements with directiveMarker
// before merging the slices. It returns a updated `original` and a patch without special elements.
// original and patch must be slices of maps, they should be checked before calling this function.
func mergeSliceWithSpecialElements(original, patch []interface{}, mergeKey string) ([]interface{}, []interface{}, error) {
	patchWithoutSpecialElements := []interface{}{}
	replace := false
	for _, v := range patch {
		typedV := v.(map[string]interface{})
		patchType, ok := typedV[directiveMarker]
		if !ok {
			patchWithoutSpecialElements = append(patchWithoutSpecialElements, v)
		} else {
			switch patchType {
			case deleteDirective:
				mergeValue, ok := typedV[mergeKey]
				if ok {
					var err error
					original, err = deleteMatchingEntries(original, mergeKey, mergeValue)
					if err != nil {
						return nil, nil, err
					}
				} else {
					return nil, nil, mergepatch.ErrNoMergeKey(typedV, mergeKey)
				}
			case replaceDirective:
				replace = true
				// Continue iterating through the array to prune any other $patch elements.
			case mergeDirective:
				return nil, nil, fmt.Errorf("merging lists cannot yet be specified in the patch")
			default:
				return nil, nil, mergepatch.ErrBadPatchType(patchType, typedV)
			}
		}
	}
	if replace {
		return patchWithoutSpecialElements, nil, nil
	}
	return original, patchWithoutSpecialElements, nil
}

// delete all matching entries (based on merge key) from a merging list
func deleteMatchingEntries(original []interface{}, mergeKey string, mergeValue interface{}) ([]interface{}, error) {
	for {
		_, originalKey, found, err := findMapInSliceBasedOnKeyValue(original, mergeKey, mergeValue)
		if err != nil {
			return nil, err
		}

		if !found {
			break
		}
		// Delete the element at originalKey.
		original = append(original[:originalKey], original[originalKey+1:]...)
	}
	return original, nil
}

// mergeSliceWithoutSpecialElements merges slices with non-special elements.
// original and patch must be slices of maps, they should be checked before calling this function.
func mergeSliceWithoutSpecialElements(original, patch []interface{}, mergeKey string, schema LookupPatchMeta, mergeOptions MergeOptions) ([]interface{}, error) {
	for _, v := range patch {
		typedV := v.(map[string]interface{})
		mergeValue, ok := typedV[mergeKey]
		if !ok {
			return nil, mergepatch.ErrNoMergeKey(typedV, mergeKey)
		}

		// If we find a value with this merge key value in original, merge the
		// maps. Otherwise append onto original.
		originalMap, originalKey, found, err := findMapInSliceBasedOnKeyValue(original, mergeKey, mergeValue)
		if err != nil {
			return nil, err
		}

		if found {
			var mergedMaps interface{}
			var err error
			// Merge into original.
			mergedMaps, err = mergeMap(originalMap, typedV, schema, mergeOptions)
			if err != nil {
				return nil, err
			}

			original[originalKey] = mergedMaps
		} else {
			original = append(original, v)
		}
	}
	return original, nil
}

// deleteFromSlice uses the parallel list to delete the items in a list of scalars
func deleteFromSlice(current, toDelete []interface{}) []interface{} {
	toDeleteMap := map[interface{}]interface{}{}
	processed := make([]interface{}, 0, len(current))
	for _, v := range toDelete {
		toDeleteMap[v] = true
	}
	for _, v := range current {
		if _, found := toDeleteMap[v]; !found {
			processed = append(processed, v)
		}
	}
	return processed
}

// This method no longer panics if any element of the slice is not a map.
func findMapInSliceBasedOnKeyValue(m []interface{}, key string, value interface{}) (map[string]interface{}, int, bool, error) {
	for k, v := range m {
		typedV, ok := v.(map[string]interface{})
		if !ok {
			return nil, 0, false, fmt.Errorf("value for key %v is not a map", k)
		}

		valueToMatch, ok := typedV[key]
		if ok && valueToMatch == value {
			return typedV, k, true, nil
		}
	}

	return nil, 0, false, nil
}
