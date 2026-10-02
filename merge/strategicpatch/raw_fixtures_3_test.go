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

var strategicMergePatchRawTestCases3 = []StrategicMergePatchRawTestCase{
	{
		Description: "add field to map in merging list by pointer with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergeItemPtr:
  - name: 1
    mergeItemPtr:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - $setElementOrder/mergeItemPtr:
      - name: 1
      - name: 2
    name: 1
    mergeItemPtr:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergeItemPtr:
  - name: 1
    mergeItemPtr:
      - name: 1
        value: 1
      - name: 2
        value: 2
  - name: 2
`),
			Current: []byte(`
mergeItemPtr:
  - name: 1
    other: a
    mergeItemPtr:
      - name: 1
        value: a
      - name: 2
        value: 2
        other: b
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - $setElementOrder/mergeItemPtr:
      - name: 1
      - name: 2
    name: 1
    mergeItemPtr:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergeItemPtr:
  - name: 1
    other: a
    mergeItemPtr:
      - name: 1
        value: 1
      - name: 2
        value: 2
        other: b
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "merge lists of scalars",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
- 1
- 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
- 1
- 2
- 3
mergingIntList:
- 3
`),
			Modified: []byte(`
mergingIntList:
- 1
- 2
- 3
`),
			Current: []byte(`
mergingIntList:
- 1
- 2
- 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
- 1
- 2
- 3
mergingIntList:
- 3
`),
			Result: []byte(`
mergingIntList:
- 1
- 2
- 3
- 4
`),
		},
	},
	{
		Description: "add duplicate field to map in merging int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
  - 3
mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
			ThreeWay: []byte(`{}`),
			Result: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
		},
	},
	// test case for setElementOrder
	{
		Description: "add an item in a list of primitives and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
- 1
- 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
- 3
- 1
- 2
mergingIntList:
- 3
`),
			Modified: []byte(`
mergingIntList:
- 3
- 1
- 2
`),
			Current: []byte(`
mergingIntList:
- 1
- 4
- 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
- 3
- 1
- 2
mergingIntList:
- 3
`),
			Result: []byte(`
mergingIntList:
- 3
- 1
- 4
- 2
`),
		},
	},
	{
		Description: "delete an item in a list of primitives and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
- 1
- 2
- 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
- 2
- 1
$deleteFromPrimitiveList/mergingIntList:
- 3
`),
			Modified: []byte(`
mergingIntList:
- 2
- 1
`),
			Current: []byte(`
mergingIntList:
- 1
- 2
- 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
- 2
- 1
$deleteFromPrimitiveList/mergingIntList:
- 3
`),
			Result: []byte(`
mergingIntList:
- 2
- 1
`),
		},
	},
	{
		Description: "add an item in a list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 3
  - name: 1
  - name: 2
mergingList:
  - name: 3
    value: 3
`),
			Modified: []byte(`
mergingList:
  - name: 3
    value: 3
  - name: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 3
  - name: 1
  - name: 2
mergingList:
  - name: 3
    value: 3
`),
			Result: []byte(`
mergingList:
  - name: 3
    value: 3
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "add multiple items in a list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 4
  - name: 2
  - name: 3
mergingList:
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
			Modified: []byte(`
mergingList:
  - name: 1
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 3
    value: 3
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 4
  - name: 2
  - name: 3
mergingList:
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 4
    value: 4
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
		},
	},
	{
		Description: "delete an item in a list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
    value: 3
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
mergingList:
  - name: 3
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 1
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
mergingList:
  - name: 3
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
    value: 2
    other: b
  - name: 1
    other: a
`),
		},
	},
	{
		Description: "change an item in a list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
    value: 3
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 3
  - name: 1
mergingList:
  - name: 3
    value: x
`),
			Modified: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 3
    value: x
  - name: 1
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 3
  - name: 1
mergingList:
  - name: 3
    value: x
`),
			Result: []byte(`
mergingList:
  - name: 2
    value: 2
    other: b
  - name: 3
    value: x
  - name: 1
    other: a
`),
		},
	},
	{
		Description: "add and delete an item in a list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
    value: 3
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 1
mergingList:
  - name: 4
    value: 4
  - name: 3
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 1
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 1
mergingList:
  - name: 4
    value: 4
  - name: 3
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
    other: b
  - name: 1
    other: a
`),
		},
	},
	{
		Description: "set elements order in a list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
    value: 3
  - name: 4
    value: 4
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 3
  - name: 1
`),
			Modified: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 1
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 3
    value: 3
  - name: 4
    value: 4
  - name: 2
    value: 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 3
  - name: 1
`),
			Result: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 1
    other: a
`),
		},
	},
	{
		Description: "set elements order in a list with server-only items",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 3
    value: 3
  - name: 4
    value: 4
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 3
  - name: 1
`),
			Modified: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 1
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 3
    value: 3
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 9
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 2
  - name: 3
  - name: 1
`),
			Result: []byte(`
mergingList:
  - name: 4
    value: 4
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 1
    other: a
  - name: 9
`),
		},
	},
	{
		Description: "set elements order in a list with server-only items 2",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 4
    value: 4
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
  - name: 4
  - name: 3
`),
			Modified: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 1
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
  - name: 9
  - name: 3
    value: 3
  - name: 4
    value: 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
  - name: 4
  - name: 3
`),
			Result: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 1
    other: a
  - name: 9
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
		},
	},
	{
		Description: "set elements order in a list with server-only items 3",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
  - name: 3
    value: 3
  - name: 4
    value: 4
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
  - name: 4
  - name: 3
`),
			Modified: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 1
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
  - name: 7
  - name: 9
  - name: 8
  - name: 3
    value: 3
  - name: 4
    value: 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 1
  - name: 4
  - name: 3
`),
			Result: []byte(`
mergingList:
  - name: 2
    value: 2
  - name: 1
    other: a
  - name: 7
  - name: 9
  - name: 8
  - name: 4
    value: 4
  - name: 3
    value: 3
`),
		},
	},
	{
		Description: "add an item in a int list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 3
  - 1
  - 2
mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 3
  - 1
  - 2
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 3
  - 1
  - 2
mergingIntList:
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 3
  - 1
  - 2
`),
		},
	},
	{
		Description: "add multiple items in a int list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 4
  - 2
  - 3
mergingIntList:
  - 4
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 4
  - 2
  - 3
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 4
  - 2
  - 3
mergingIntList:
  - 4
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 1
  - 4
  - 2
  - 3
`),
		},
	},
}
