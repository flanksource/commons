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
	"strings"

	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// handleDirectiveInMergeMap handles the patch directive when merging 2 maps.
func handleDirectiveInMergeMap(directive interface{}, patch map[string]interface{}) (map[string]interface{}, error) {
	if directive == replaceDirective {
		// If the patch contains "$patch: replace", don't merge it, just use the
		// patch directly. Later on, we can add a single level replace that only
		// affects the map that the $patch is in.
		delete(patch, directiveMarker)
		return patch, nil
	}

	if directive == deleteDirective {
		// If the patch contains "$patch: delete", don't merge it, just return
		//  an empty map.
		return map[string]interface{}{}, nil
	}

	return nil, mergepatch.ErrBadPatchType(directive, patch)
}

func containsDirectiveMarker(item interface{}) bool {
	m, ok := item.(map[string]interface{})
	if ok {
		if _, foundDirectiveMarker := m[directiveMarker]; foundDirectiveMarker {
			return true
		}
	}
	return false
}

func mergeKeyValueEqual(left, right interface{}, mergeKey string) (bool, error) {
	if len(mergeKey) == 0 {
		return left == right, nil
	}
	typedLeft, ok := left.(map[string]interface{})
	if !ok {
		return false, mergepatch.ErrBadArgType(typedLeft, left)
	}
	typedRight, ok := right.(map[string]interface{})
	if !ok {
		return false, mergepatch.ErrBadArgType(typedRight, right)
	}
	mergeKeyLeft, ok := typedLeft[mergeKey]
	if !ok {
		return false, mergepatch.ErrNoMergeKey(typedLeft, mergeKey)
	}
	mergeKeyRight, ok := typedRight[mergeKey]
	if !ok {
		return false, mergepatch.ErrNoMergeKey(typedRight, mergeKey)
	}
	return mergeKeyLeft == mergeKeyRight, nil
}

// extractKey trims the prefix and return the original key
func extractKey(s, prefix string) (string, error) {
	substrings := strings.SplitN(s, "/", 2)
	if len(substrings) <= 1 || substrings[0] != prefix {
		switch prefix {
		case deleteFromPrimitiveListDirectivePrefix:
			return "", mergepatch.ErrBadPatchFormatForPrimitiveList
		case setElementOrderDirectivePrefix:
			return "", mergepatch.ErrBadPatchFormatForSetElementOrderList
		default:
			return "", fmt.Errorf("fail to find unknown prefix %q in %s", prefix, s)
		}
	}
	return substrings[1], nil
}

// validatePatchUsingSetOrderList verifies:
// the relative order of any two items in the setOrderList list matches that in the patch list.
// the items in the patch list must be a subset or the same as the $setElementOrder list (deletions are ignored).
func validatePatchWithSetOrderList(patchList, setOrderList interface{}, mergeKey string) error {
	typedSetOrderList, ok := setOrderList.([]interface{})
	if !ok {
		return mergepatch.ErrBadPatchFormatForSetElementOrderList
	}
	typedPatchList, ok := patchList.([]interface{})
	if !ok {
		return mergepatch.ErrBadPatchFormatForSetElementOrderList
	}
	if len(typedSetOrderList) == 0 || len(typedPatchList) == 0 {
		return nil
	}

	var nonDeleteList []interface{}
	var err error
	if len(mergeKey) > 0 {
		nonDeleteList, _, err = extractToDeleteItems(typedPatchList)
		if err != nil {
			return err
		}
	} else {
		nonDeleteList = typedPatchList
	}

	patchIndex, setOrderIndex := 0, 0
	for patchIndex < len(nonDeleteList) && setOrderIndex < len(typedSetOrderList) {
		if containsDirectiveMarker(nonDeleteList[patchIndex]) {
			patchIndex++
			continue
		}
		mergeKeyEqual, err := mergeKeyValueEqual(nonDeleteList[patchIndex], typedSetOrderList[setOrderIndex], mergeKey)
		if err != nil {
			return err
		}
		if mergeKeyEqual {
			patchIndex++
		}
		setOrderIndex++
	}
	// If patchIndex is inbound but setOrderIndex if out of bound mean there are items mismatching between the patch list and setElementOrder list.
	// the second check is a sanity check, and should always be true if the first is true.
	if patchIndex < len(nonDeleteList) && setOrderIndex >= len(typedSetOrderList) {
		return fmt.Errorf("the order in patch list:\n%v\n doesn't match %s list:\n%v", typedPatchList, setElementOrderDirectivePrefix, setOrderList)
	}
	return nil
}

// preprocessDeletionListForMerging preprocesses the deletion list.
// it returns shouldContinue, isDeletionList, noPrefixKey
func preprocessDeletionListForMerging(key string, original map[string]interface{},
	patchVal interface{}, mergeDeletionList bool) (bool, bool, string, error) {
	// If found a parallel list for deletion and we are going to merge the list,
	// overwrite the key to the original key and set flag isDeleteList
	foundParallelListPrefix := strings.HasPrefix(key, deleteFromPrimitiveListDirectivePrefix)
	if foundParallelListPrefix {
		if !mergeDeletionList {
			original[key] = patchVal
			return true, false, "", nil
		}
		originalKey, err := extractKey(key, deleteFromPrimitiveListDirectivePrefix)
		return false, true, originalKey, err
	}
	return false, false, "", nil
}

// applyRetainKeysDirective looks for a retainKeys directive and applies to original
// - if no directive exists do nothing
// - if directive is found, clear keys in original missing from the directive list
// - validate that all keys present in the patch are present in the retainKeys directive
// note: original may be another patch request, e.g. applying the add+modified patch to the deletions patch. In this case it may have directives
func applyRetainKeysDirective(original, patch map[string]interface{}, options MergeOptions) error {
	retainKeysInPatch, foundInPatch := patch[retainKeysDirective]
	if !foundInPatch {
		return nil
	}
	// cleanup the directive
	delete(patch, retainKeysDirective)

	if !options.MergeParallelList {
		// If original is actually a patch, make sure the retainKeys directives are the same in both patches if present in both.
		// If not present in the original patch, copy from the modified patch.
		retainKeysInOriginal, foundInOriginal := original[retainKeysDirective]
		if foundInOriginal {
			if !reflect.DeepEqual(retainKeysInOriginal, retainKeysInPatch) {
				// This error actually should never happen.
				return fmt.Errorf("%v and %v are not deep equal: this may happen when calculating the 3-way diff patch", retainKeysInOriginal, retainKeysInPatch)
			}
		} else {
			original[retainKeysDirective] = retainKeysInPatch
		}
		return nil
	}

	retainKeysList, ok := retainKeysInPatch.([]interface{})
	if !ok {
		return mergepatch.ErrBadPatchFormatForRetainKeys
	}

	// validate patch to make sure all fields in the patch are present in the retainKeysList.
	// The map is used only as a set, the value is never referenced
	m := map[interface{}]struct{}{}
	for _, v := range retainKeysList {
		m[v] = struct{}{}
	}
	for k, v := range patch {
		if v == nil || strings.HasPrefix(k, deleteFromPrimitiveListDirectivePrefix) ||
			strings.HasPrefix(k, setElementOrderDirectivePrefix) {
			continue
		}
		// If there is an item present in the patch but not in the retainKeys list,
		// the patch is invalid.
		if _, found := m[k]; !found {
			return mergepatch.ErrBadPatchFormatForRetainKeys
		}
	}

	// clear not present fields
	for k := range original {
		if _, found := m[k]; !found {
			delete(original, k)
		}
	}
	return nil
}

// mergePatchIntoOriginal processes $setElementOrder list.
// When not merging the directive, it will make sure $setElementOrder list exist only in original.
// When merging the directive, it will try to find the $setElementOrder list and
// its corresponding patch list, validate it and merge it.
// Then, sort them by the relative order in setElementOrder, patch list and live list.
// The precedence is $setElementOrder > order in patch list > order in live list.
// This function will delete the item after merging it to prevent process it again in the future.
// Ref: https://git.k8s.io/design-proposals-archive/cli/preserve-order-in-strategic-merge-patch.md
func mergePatchIntoOriginal(original, patch map[string]interface{}, schema LookupPatchMeta, mergeOptions MergeOptions) error {
	for key, patchV := range patch {
		// Do nothing if there is no ordering directive
		if !strings.HasPrefix(key, setElementOrderDirectivePrefix) {
			continue
		}

		setElementOrderInPatch := patchV
		// Copies directive from the second patch (`patch`) to the first patch (`original`)
		// and checks they are equal and delete the directive in the second patch
		if !mergeOptions.MergeParallelList {
			setElementOrderListInOriginal, ok := original[key]
			if ok {
				// check if the setElementOrder list in original and the one in patch matches
				if !reflect.DeepEqual(setElementOrderListInOriginal, setElementOrderInPatch) {
					return mergepatch.ErrBadPatchFormatForSetElementOrderList
				}
			} else {
				// move the setElementOrder list from patch to original
				original[key] = setElementOrderInPatch
			}
		}
		delete(patch, key)

		var (
			ok                                          bool
			originalFieldValue, patchFieldValue, merged []interface{}
			patchStrategy                               string
			patchMeta                                   PatchMeta
			subschema                                   LookupPatchMeta
		)
		typedSetElementOrderList, ok := setElementOrderInPatch.([]interface{})
		if !ok {
			return mergepatch.ErrBadArgType(typedSetElementOrderList, setElementOrderInPatch)
		}
		// Trim the setElementOrderDirectivePrefix to get the key of the list field in original.
		originalKey, err := extractKey(key, setElementOrderDirectivePrefix)
		if err != nil {
			return err
		}
		// try to find the list with `originalKey` in `original` and `modified` and merge them.
		originalList, foundOriginal := original[originalKey]
		patchList, foundPatch := patch[originalKey]
		if foundOriginal {
			originalFieldValue, ok = originalList.([]interface{})
			if !ok {
				return mergepatch.ErrBadArgType(originalFieldValue, originalList)
			}
		}
		if foundPatch {
			patchFieldValue, ok = patchList.([]interface{})
			if !ok {
				return mergepatch.ErrBadArgType(patchFieldValue, patchList)
			}
		}
		subschema, patchMeta, err = schema.LookupPatchMetadataForSlice(originalKey)
		if err != nil {
			return err
		}
		_, patchStrategy, err = extractRetainKeysPatchStrategy(patchMeta.GetPatchStrategies())
		if err != nil {
			return err
		}
		// Check for consistency between the element order list and the field it applies to
		err = validatePatchWithSetOrderList(patchFieldValue, typedSetElementOrderList, patchMeta.GetPatchMergeKey())
		if err != nil {
			return err
		}

		switch {
		case foundOriginal && !foundPatch:
			// no change to list contents
			merged = originalFieldValue
		case !foundOriginal && foundPatch:
			// list was added
			v, keep := removeDirectives(patchFieldValue)
			if !keep {
				// Shouldn't be possible since patchFieldValue is a slice
				continue
			}

			merged = v.([]interface{})
		case foundOriginal && foundPatch:
			merged, err = mergeSliceHandler(originalList, patchList, subschema,
				patchStrategy, patchMeta.GetPatchMergeKey(), false, mergeOptions)
			if err != nil {
				return err
			}
		case !foundOriginal && !foundPatch:
			continue
		}

		// Split all items into patch items and server-only items and then enforce the order.
		var patchItems, serverOnlyItems []interface{}
		if len(patchMeta.GetPatchMergeKey()) == 0 {
			// Primitives doesn't need merge key to do partitioning.
			patchItems, serverOnlyItems = partitionPrimitivesByPresentInList(merged, typedSetElementOrderList)

		} else {
			// Maps need merge key to do partitioning.
			patchItems, serverOnlyItems, err = partitionMapsByPresentInList(merged, typedSetElementOrderList, patchMeta.GetPatchMergeKey())
			if err != nil {
				return err
			}
		}

		elementType, err := sliceElementType(originalFieldValue, patchFieldValue)
		if err != nil {
			return err
		}
		kind := elementType.Kind()
		// normalize merged list
		// typedSetElementOrderList contains all the relative order in typedPatchList,
		// so don't need to use typedPatchList
		both, err := normalizeElementOrder(patchItems, serverOnlyItems, typedSetElementOrderList, originalFieldValue, patchMeta.GetPatchMergeKey(), kind)
		if err != nil {
			return err
		}
		original[originalKey] = both
		// delete patch list from patch to prevent process again in the future
		delete(patch, originalKey)
	}
	return nil
}

// partitionPrimitivesByPresentInList partitions elements into 2 slices, the first containing items present in partitionBy, the other not.
func partitionPrimitivesByPresentInList(original, partitionBy []interface{}) ([]interface{}, []interface{}) {
	patch := make([]interface{}, 0, len(original))
	serverOnly := make([]interface{}, 0, len(original))
	inPatch := map[interface{}]bool{}
	for _, v := range partitionBy {
		inPatch[v] = true
	}
	for _, v := range original {
		if !inPatch[v] {
			serverOnly = append(serverOnly, v)
		} else {
			patch = append(patch, v)
		}
	}
	return patch, serverOnly
}

// partitionMapsByPresentInList partitions elements into 2 slices, the first containing items present in partitionBy, the other not.
func partitionMapsByPresentInList(original, partitionBy []interface{}, mergeKey string) ([]interface{}, []interface{}, error) {
	patch := make([]interface{}, 0, len(original))
	serverOnly := make([]interface{}, 0, len(original))
	for _, v := range original {
		typedV, ok := v.(map[string]interface{})
		if !ok {
			return nil, nil, mergepatch.ErrBadArgType(typedV, v)
		}
		mergeKeyValue, foundMergeKey := typedV[mergeKey]
		if !foundMergeKey {
			return nil, nil, mergepatch.ErrNoMergeKey(typedV, mergeKey)
		}
		_, _, found, err := findMapInSliceBasedOnKeyValue(partitionBy, mergeKey, mergeKeyValue)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			serverOnly = append(serverOnly, v)
		} else {
			patch = append(patch, v)
		}
	}
	return patch, serverOnly, nil
}

// Removes directives from an object and returns value to use instead and whether
// or not the field/index should even be kept
// May modify input
func removeDirectives(obj interface{}) (interface{}, bool) {
	if obj == nil {
		return obj, true
	} else if typedV, ok := obj.(map[string]interface{}); ok {
		if _, hasDirective := typedV[directiveMarker]; hasDirective {
			return nil, false
		}

		for k, v := range typedV {
			var keep bool
			typedV[k], keep = removeDirectives(v)
			if !keep {
				delete(typedV, k)
			}
		}
		return typedV, true
	} else if typedV, ok := obj.([]interface{}); ok {
		var res []interface{}
		if typedV != nil {
			// Make sure res is non-nil if patch is non-nil
			res = []interface{}{}
		}
		for _, v := range typedV {
			if newV, keep := removeDirectives(v); keep {
				res = append(res, newV)
			}
		}
		return res, true
	} else {
		return obj, true
	}
}
