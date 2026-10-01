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

	"github.com/flanksource/commons/merge/strategicpatch/internal/json"
	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// An alternate implementation of JSON Merge Patch
// (https://tools.ietf.org/html/rfc7386) which supports the ability to annotate
// certain fields with metadata that indicates whether the elements of JSON
// lists should be merged or replaced.
//
// For more information, see the PATCH section of docs/devel/api-conventions.md.
//
// Some of the content of this package was borrowed with minor adaptations from
// evanphx/json-patch and openshift/origin.

const (
	directiveMarker  = "$patch"
	deleteDirective  = "delete"
	replaceDirective = "replace"
	mergeDirective   = "merge"

	retainKeysStrategy = "retainKeys"

	deleteFromPrimitiveListDirectivePrefix = "$deleteFromPrimitiveList"
	retainKeysDirective                    = "$" + retainKeysStrategy
	setElementOrderDirectivePrefix         = "$setElementOrder"
)

// JSONMap is a representations of JSON object encoded as map[string]interface{}
// where the children can be either map[string]interface{}, []interface{} or
// primitive type).
// Operating on JSONMap representation is much faster as it doesn't require any
// json marshaling and/or unmarshaling operations.
type JSONMap map[string]interface{}

type MergeOptions struct {
	// MergeParallelList indicates if we are merging the parallel list.
	// We don't merge parallel list when calling mergeMap() in CreateThreeWayMergePatch()
	// which is called client-side.
	// We merge parallel list iff when calling mergeMap() in StrategicMergeMapPatch()
	// which is called server-side
	MergeParallelList bool
	// IgnoreUnmatchedNulls indicates if we should process the unmatched nulls.
	IgnoreUnmatchedNulls bool
}

// StrategicMergePatch applies a strategic merge patch. The patch and the original document
// must be json encoded content. A patch can be created from an original and a modified document
// by calling CreateStrategicMergePatch.
func StrategicMergePatch(original, patch []byte, dataStruct interface{}) ([]byte, error) {
	schema, err := NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}

	return StrategicMergePatchUsingLookupPatchMeta(original, patch, schema)
}

func StrategicMergePatchUsingLookupPatchMeta(original, patch []byte, schema LookupPatchMeta) ([]byte, error) {
	originalMap, err := handleUnmarshal(original)
	if err != nil {
		return nil, err
	}
	patchMap, err := handleUnmarshal(patch)
	if err != nil {
		return nil, err
	}

	result, err := StrategicMergeMapPatchUsingLookupPatchMeta(originalMap, patchMap, schema)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func handleUnmarshal(j []byte) (map[string]interface{}, error) {
	if j == nil {
		j = []byte("{}")
	}

	m := map[string]interface{}{}
	err := json.Unmarshal(j, &m)
	if err != nil {
		return nil, mergepatch.ErrBadJSONDoc
	}
	return m, nil
}

// StrategicMergeMapPatch applies a strategic merge patch. The original and patch documents
// must be JSONMap. A patch can be created from an original and modified document by
// calling CreateTwoWayMergeMapPatch.
// Warning: the original and patch JSONMap objects are mutated by this function and should not be reused.
func StrategicMergeMapPatch(original, patch JSONMap, dataStruct interface{}) (JSONMap, error) {
	schema, err := NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}

	// We need the go struct tags `patchMergeKey` and `patchStrategy` for fields that support a strategic merge patch.
	// For native resources, we can easily figure out these tags since we know the fields.

	// Because custom resources are decoded as Unstructured and because we're missing the metadata about how to handle
	// each field in a strategic merge patch, we can't find the go struct tags. Hence, we can't easily  do a strategic merge
	// for custom resources. So we should fail fast and return an error.
	if isUnstructured(dataStruct) {
		return nil, mergepatch.ErrUnsupportedStrategicMergePatchFormat
	}

	return StrategicMergeMapPatchUsingLookupPatchMeta(original, patch, schema)
}

// isUnstructured reports whether dataStruct is a *unstructured.Unstructured,
// matched by type identity instead of an import so commons keeps no
// dependency on k8s.io/apimachinery.
func isUnstructured(dataStruct interface{}) bool {
	t := reflect.TypeOf(dataStruct)
	return t != nil && t.Kind() == reflect.Pointer &&
		t.Elem().PkgPath() == "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured" &&
		t.Elem().Name() == "Unstructured"
}

func StrategicMergeMapPatchUsingLookupPatchMeta(original, patch JSONMap, schema LookupPatchMeta) (JSONMap, error) {
	mergeOptions := MergeOptions{
		MergeParallelList:    true,
		IgnoreUnmatchedNulls: true,
	}
	return mergeMap(original, patch, schema, mergeOptions)
}

// MergeStrategicMergeMapPatchUsingLookupPatchMeta merges strategic merge
// patches retaining `null` fields and parallel lists. If 2 patches change the
// same fields and the latter one will override the former one. If you don't
// want that happen, you need to run func MergingMapsHaveConflicts before
// merging these patches. Applying the resulting merged merge patch to a JSONMap
// yields the same as merging each strategic merge patch to the JSONMap in
// succession.
func MergeStrategicMergeMapPatchUsingLookupPatchMeta(schema LookupPatchMeta, patches ...JSONMap) (JSONMap, error) {
	mergeOptions := MergeOptions{
		MergeParallelList:    false,
		IgnoreUnmatchedNulls: false,
	}
	merged := JSONMap{}
	var err error
	for _, patch := range patches {
		merged, err = mergeMap(merged, patch, schema, mergeOptions)
		if err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func mapTypeAssertion(original, patch interface{}) (map[string]interface{}, map[string]interface{}, error) {
	typedOriginal, ok := original.(map[string]interface{})
	if !ok {
		return nil, nil, mergepatch.ErrBadArgType(typedOriginal, original)
	}
	typedPatch, ok := patch.(map[string]interface{})
	if !ok {
		return nil, nil, mergepatch.ErrBadArgType(typedPatch, patch)
	}
	return typedOriginal, typedPatch, nil
}

func sliceTypeAssertion(original, patch interface{}) ([]interface{}, []interface{}, error) {
	typedOriginal, ok := original.([]interface{})
	if !ok {
		return nil, nil, mergepatch.ErrBadArgType(typedOriginal, original)
	}
	typedPatch, ok := patch.([]interface{})
	if !ok {
		return nil, nil, mergepatch.ErrBadArgType(typedPatch, patch)
	}
	return typedOriginal, typedPatch, nil
}

// extractRetainKeysPatchStrategy process patch strategy, which is a string may contains multiple
// patch strategies separated by ",". It returns a boolean var indicating if it has
// retainKeys strategies and a string for the other strategy.
func extractRetainKeysPatchStrategy(strategies []string) (bool, string, error) {
	switch len(strategies) {
	case 0:
		return false, "", nil
	case 1:
		singleStrategy := strategies[0]
		switch singleStrategy {
		case retainKeysStrategy:
			return true, "", nil
		default:
			return false, singleStrategy, nil
		}
	case 2:
		switch {
		case strategies[0] == retainKeysStrategy:
			return true, strategies[1], nil
		case strategies[1] == retainKeysStrategy:
			return true, strategies[0], nil
		default:
			return false, "", fmt.Errorf("unexpected patch strategy: %v", strategies)
		}
	default:
		return false, "", fmt.Errorf("unexpected patch strategy: %v", strategies)
	}
}
