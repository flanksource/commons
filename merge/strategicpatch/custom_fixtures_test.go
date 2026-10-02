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
// Ported from k8s.io/apimachinery v0.36.2 pkg/util/strategicpatch/patch_test.go; apply half only.

package strategicpatch

// These are test cases for SortMergeList, used to assert that it (recursively)
// sorts both merging and non merging lists correctly.
var sortMergeListTestCaseData = []byte(`
testCases:
  - description: sort one list of maps
    original:
      mergingList:
        - name: 1
        - name: 3
        - name: 2
    sorted:
      mergingList:
        - name: 1
        - name: 2
        - name: 3
  - description: sort lists of maps but not nested lists of maps
    original:
      mergingList:
        - name: 2
          nonMergingList:
            - name: 1
            - name: 3
            - name: 2
        - name: 1
          nonMergingList:
            - name: 2
            - name: 1
    sorted:
      mergingList:
        - name: 1
          nonMergingList:
            - name: 2
            - name: 1
        - name: 2
          nonMergingList:
            - name: 1
            - name: 3
            - name: 2
  - description: sort lists of maps and nested lists of maps
    original:
      mergingList:
        - name: 2
          mergingList:
            - name: 1
            - name: 3
            - name: 2
        - name: 1
          mergingList:
            - name: 2
            - name: 1
    sorted:
      mergingList:
        - name: 1
          mergingList:
            - name: 1
            - name: 2
        - name: 2
          mergingList:
            - name: 1
            - name: 2
            - name: 3
  - description: merging list should NOT sort when nested in non merging list
    original:
      nonMergingList:
        - name: 2
          mergingList:
            - name: 1
            - name: 3
            - name: 2
        - name: 1
          mergingList:
            - name: 2
            - name: 1
    sorted:
      nonMergingList:
        - name: 2
          mergingList:
            - name: 1
            - name: 3
            - name: 2
        - name: 1
          mergingList:
            - name: 2
            - name: 1
  - description: sort very nested list of maps
    fieldTypes:
    original:
      mergingList:
        - mergingList:
            - mergingList:
                - name: 2
                - name: 1
    sorted:
      mergingList:
        - mergingList:
            - mergingList:
                - name: 1
                - name: 2
  - description: sort nested lists of ints
    original:
      mergingList:
        - name: 2
          mergingIntList:
            - 1
            - 3
            - 2
        - name: 1
          mergingIntList:
            - 2
            - 1
    sorted:
      mergingList:
        - name: 1
          mergingIntList:
            - 1
            - 2
        - name: 2
          mergingIntList:
            - 1
            - 2
            - 3
  - description: sort nested pointers of ints
    original:
      mergeItemPtr:
        - name: 2
          mergingIntList:
            - 1
            - 3
            - 2
        - name: 1
          mergingIntList:
            - 2
            - 1
    sorted:
      mergeItemPtr:
        - name: 1
          mergingIntList:
            - 1
            - 2
        - name: 2
          mergingIntList:
            - 1
            - 2
            - 3
  - description: sort merging list by pointer
    original:
      mergeItemPtr:
        - name: 1
        - name: 3
        - name: 2
    sorted:
      mergeItemPtr:
        - name: 1
        - name: 2
        - name: 3
`)

// These are test cases for StrategicMergePatch that cannot be generated using
// CreateTwoWayMergePatch because it may be one of the following cases:
// - not use the replace directive.
// - generate duplicate integers for a merging list patch.
// - generate empty merging lists.
// - use patch format from an old client.
var customStrategicMergePatchTestCaseData = []byte(`
testCases:
  - description: unique scalars when merging lists
    original:
      mergingIntList:
        - 1
        - 2
    twoWay:
      mergingIntList:
        - 2
        - 3
    modified:
      mergingIntList:
        - 1
        - 2
        - 3
  - description: delete map from nested map
    original:
      simpleMap:
        key1: 1
        key2: 1
    twoWay:
      simpleMap:
        $patch: delete
    modified:
      simpleMap:
        {}
  - description: delete all items from merging list
    original:
      mergingList:
        - name: 1
        - name: 2
    twoWay:
      mergingList:
        - $patch: replace
    modified:
      mergingList: []
  - description: merge empty merging lists
    original:
      mergingList: []
    twoWay:
      mergingList: []
    modified:
      mergingList: []
  - description: delete all keys from map
    original:
      name: 1
      value: 1
    twoWay:
      $patch: replace
    modified: {}
  - description: add key and delete all keys from map
    original:
      name: 1
      value: 1
    twoWay:
      other: a
      $patch: replace
    modified:
      other: a
  - description: delete all duplicate entries in a merging list
    original:
      mergingList:
        - name: 1
        - name: 1
        - name: 2
          value: a
        - name: 3
        - name: 3
    twoWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 3
          $patch: delete
    modified:
      mergingList:
        - name: 2
          value: a
  - description: retainKeys map can add a field when no retainKeys directive present
    original:
      retainKeysMap:
        name: foo
    twoWay:
      retainKeysMap:
        value: bar
    modified:
      retainKeysMap:
        name: foo
        value: bar
  - description: retainKeys map can change a field when no retainKeys directive present
    original:
      retainKeysMap:
        name: foo
        value: a
    twoWay:
      retainKeysMap:
        value: b
    modified:
      retainKeysMap:
        name: foo
        value: b
  - description: retainKeys map can delete a field when no retainKeys directive present
    original:
      retainKeysMap:
        name: foo
        value: a
    twoWay:
      retainKeysMap:
        value: null
    modified:
      retainKeysMap:
        name: foo
  - description: retainKeys map merge an empty map
    original:
      retainKeysMap:
        name: foo
        value: a
    twoWay:
      retainKeysMap: {}
    modified:
      retainKeysMap:
        name: foo
        value: a
  - description: retainKeys list can add a field when no retainKeys directive present
    original:
      retainKeysMergingList:
      - name: bar
      - name: foo
    twoWay:
      retainKeysMergingList:
      - name: foo
        value: a
    modified:
      retainKeysMergingList:
      - name: bar
      - name: foo
        value: a
  - description: retainKeys list can change a field when no retainKeys directive present
    original:
      retainKeysMergingList:
      - name: bar
      - name: foo
        value: a
    twoWay:
      retainKeysMergingList:
      - name: foo
        value: b
    modified:
      retainKeysMergingList:
      - name: bar
      - name: foo
        value: b
  - description: retainKeys list can delete a field when no retainKeys directive present
    original:
      retainKeysMergingList:
      - name: bar
      - name: foo
        value: a
    twoWay:
      retainKeysMergingList:
      - name: foo
        value: null
    modified:
      retainKeysMergingList:
      - name: bar
      - name: foo
  - description: preserve the order from the patch in a merging list
    original:
      mergingList:
        - name: 1
        - name: 2
          value: b
        - name: 3
    twoWay:
      mergingList:
        - name: 3
          value: c
        - name: 1
          value: a
        - name: 2
          other: x
    modified:
      mergingList:
        - name: 3
          value: c
        - name: 1
          value: a
        - name: 2
          value: b
          other: x
  - description: preserve the order from the patch in a merging list 2
    original:
      mergingList:
        - name: 1
        - name: 2
          value: b
        - name: 3
    twoWay:
      mergingList:
        - name: 3
          value: c
        - name: 1
          value: a
    modified:
      mergingList:
        - name: 2
          value: b
        - name: 3
          value: c
        - name: 1
          value: a
  - description: preserve the order from the patch in a merging int list
    original:
      mergingIntList:
        - 1
        - 2
        - 3
    twoWay:
      mergingIntList:
        - 3
        - 1
        - 2
    modified:
      mergingIntList:
        - 3
        - 1
        - 2
  - description: preserve the order from the patch in a merging int list
    original:
      mergingIntList:
        - 1
        - 2
        - 3
    twoWay:
      mergingIntList:
        - 3
        - 1
    modified:
      mergingIntList:
        - 2
        - 3
        - 1
`)

var customStrategicMergePatchRawTestCases = []StrategicMergePatchRawTestCase{
	{
		Description: "$setElementOrder contains item that is not present in the list to be merged",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 3
  - name: 2
  - name: 1
mergingList:
  - name: 3
    value: 3
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 3
    value: 3
  - name: 1
    value: 1
`),
		},
	},
	{
		Description: "$setElementOrder contains item that is not present in the int list to be merged",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 3
  - 2
  - 1
`),
			Modified: []byte(`
mergingIntList:
  - 3
  - 1
`),
		},
	},
	{
		Description: "should check if order in $setElementOrder and patch list match",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
  - name: 3
mergingList:
  - name: 3
    value: 3
  - name: 1
    value: 1
`),
			ExpectedError: "doesn't match",
		},
	},
	{
		Description: "$setElementOrder contains item that is not present in the int list to be merged",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
  - 3
mergingIntList:
  - 3
  - 1
`),
			ExpectedError: "doesn't match",
		},
	},
	{
		Description: "missing merge key should error out",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: a
`),
			TwoWay: []byte(`
mergingList:
  - value: b
`),
			ExpectedError: "does not contain declared merge key",
		},
	},
	{
		Description: "$deleteFromPrimitiveList of nonexistent item in primitive list should not add the item to the list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
`),
			TwoWay: []byte(`
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 2
`),
		},
	},
	{
		Description: "$deleteFromPrimitiveList on empty primitive list should not add the item to the list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
`),
			TwoWay: []byte(`
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
`),
		},
	},
	{
		Description: "$deleteFromPrimitiveList on nonexistent primitive list should not add the primitive list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
foo:
  - bar
`),
			TwoWay: []byte(`
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
foo:
  - bar
`),
		},
	},
	{
		Description: "$deleteFromPrimitiveList should delete item from a list with merge patch strategy",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
			TwoWay: []byte(`
$deleteFromPrimitiveList/mergingIntList:
  - 2
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 3
`),
		},
	},
	{
		Description: "$deleteFromPrimitiveList should delete item from a list without merge patch strategy",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
nonMergingIntList:
  - 1
  - 2
  - 3
`),
			TwoWay: []byte(`
$deleteFromPrimitiveList/nonMergingIntList:
  - 2
`),
			Modified: []byte(`
nonMergingIntList:
  - 1
  - 3
`),
		},
	},
}
