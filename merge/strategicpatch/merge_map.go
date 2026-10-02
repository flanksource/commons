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
	"reflect"
)

// Merge fields from a patch map into the original map. Note: This may modify
// both the original map and the patch because getting a deep copy of a map in
// golang is highly non-trivial.
// flag mergeOptions.MergeParallelList controls if using the parallel list to delete or keeping the list.
// If patch contains any null field (e.g. field_1: null) that is not
// present in original, then to propagate it to the end result use
// mergeOptions.IgnoreUnmatchedNulls == false.
func mergeMap(original, patch map[string]interface{}, schema LookupPatchMeta, mergeOptions MergeOptions) (map[string]interface{}, error) {
	if v, ok := patch[directiveMarker]; ok {
		return handleDirectiveInMergeMap(v, patch)
	}

	// nil is an accepted value for original to simplify logic in other places.
	// If original is nil, replace it with an empty map and then apply the patch.
	if original == nil {
		original = map[string]interface{}{}
	}

	err := applyRetainKeysDirective(original, patch, mergeOptions)
	if err != nil {
		return nil, err
	}

	// Process $setElementOrder list and other lists sharing the same key.
	// When not merging the directive, it will make sure $setElementOrder list exist only in original.
	// When merging the directive, it will process $setElementOrder and its patch list together.
	// This function will delete the merged elements from patch so they will not be reprocessed
	err = mergePatchIntoOriginal(original, patch, schema, mergeOptions)
	if err != nil {
		return nil, err
	}

	// Start merging the patch into the original.
	for k, patchV := range patch {
		skipProcessing, isDeleteList, noPrefixKey, err := preprocessDeletionListForMerging(k, original, patchV, mergeOptions.MergeParallelList)
		if err != nil {
			return nil, err
		}
		if skipProcessing {
			continue
		}
		if len(noPrefixKey) > 0 {
			k = noPrefixKey
		}

		// If the value of this key is null, delete the key if it exists in the
		// original. Otherwise, check if we want to preserve it or skip it.
		// Preserving the null value is useful when we want to send an explicit
		// delete to the API server.
		// In some cases, this may lead to inconsistent behavior with create.
		// ref: https://github.com/kubernetes/kubernetes/issues/123304
		// To avoid breaking compatibility,
		// we made corresponding changes on the client side to ensure that the create and patch behaviors are idempotent.
		if patchV == nil {
			delete(original, k)
			if mergeOptions.IgnoreUnmatchedNulls {
				continue
			}
		}

		_, ok := original[k]
		if !ok {
			if !isDeleteList {
				// If it's not in the original document, just take the patch value.
				if mergeOptions.IgnoreUnmatchedNulls {
					discardNullValuesFromPatch(patchV)
				}
				original[k], ok = removeDirectives(patchV)
				if !ok {
					delete(original, k)
				}
			}
			continue
		}

		originalType := reflect.TypeOf(original[k])
		patchType := reflect.TypeOf(patchV)
		if originalType != patchType {
			if !isDeleteList {
				if mergeOptions.IgnoreUnmatchedNulls {
					discardNullValuesFromPatch(patchV)
				}
				original[k], ok = removeDirectives(patchV)
				if !ok {
					delete(original, k)
				}
			}
			continue
		}
		// If they're both maps or lists, recurse into the value.
		switch originalType.Kind() {
		case reflect.Map:
			subschema, patchMeta, err2 := schema.LookupPatchMetadataForStruct(k)
			if err2 != nil {
				return nil, err2
			}
			_, patchStrategy, err2 := extractRetainKeysPatchStrategy(patchMeta.GetPatchStrategies())
			if err2 != nil {
				return nil, err2
			}
			original[k], err = mergeMapHandler(original[k], patchV, subschema, patchStrategy, mergeOptions)
		case reflect.Slice:
			subschema, patchMeta, err2 := schema.LookupPatchMetadataForSlice(k)
			if err2 != nil {
				return nil, err2
			}
			_, patchStrategy, err2 := extractRetainKeysPatchStrategy(patchMeta.GetPatchStrategies())
			if err2 != nil {
				return nil, err2
			}
			original[k], err = mergeSliceHandler(original[k], patchV, subschema, patchStrategy, patchMeta.GetPatchMergeKey(), isDeleteList, mergeOptions)
		default:
			original[k], ok = removeDirectives(patchV)
			if !ok {
				// if patchV itself is a directive, then don't keep it
				delete(original, k)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return original, nil
}

// discardNullValuesFromPatch discards all null property values from patch.
// It traverses all slices and map types.
func discardNullValuesFromPatch(patchV interface{}) {
	switch patchV := patchV.(type) {
	case map[string]interface{}:
		for k, v := range patchV {
			if v == nil {
				delete(patchV, k)
			} else {
				discardNullValuesFromPatch(v)
			}
		}
	case []interface{}:
		for _, v := range patchV {
			discardNullValuesFromPatch(v)
		}
	}
}

// mergeMapHandler handles how to merge `patchV` whose key is `key` with `original` respecting
// fieldPatchStrategy and mergeOptions.
func mergeMapHandler(original, patch interface{}, schema LookupPatchMeta,
	fieldPatchStrategy string, mergeOptions MergeOptions) (map[string]interface{}, error) {
	typedOriginal, typedPatch, err := mapTypeAssertion(original, patch)
	if err != nil {
		return nil, err
	}

	if fieldPatchStrategy != replaceDirective {
		return mergeMap(typedOriginal, typedPatch, schema, mergeOptions)
	} else {
		return typedPatch, nil
	}
}
