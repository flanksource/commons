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

var strategicMergePatchRawTestCases2 = []StrategicMergePatchRawTestCase{
	{
		Description: "add map and delete map from merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 3
mergingList:
  - name: 3
  - name: 1
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
  - name: 3
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 4
    other: c
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 3
mergingList:
  - name: 2
  - name: 3
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 4
    other: c
  - name: 2
  - name: 3
`),
		},
	},
	{
		Description: "delete field from map in merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Modified: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "delete field from map in merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Modified: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    value: a
    other: a
  - name: 2
    value: 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
`),
		},
	},
	{
		Description: "delete missing field from map in merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Modified: []byte(`
mergingList:
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
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "delete missing field from map in merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
`),
			Modified: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: null
  - name: 2
    value: 2
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "replace non merging list nested in merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "replace non merging list nested in merging list with value conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 1
        value: c
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "replace non merging list nested in merging list with deletion conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 2
        value: 2
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    nonMergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    nonMergingList:
      - name: 1
        value: 1
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "add field to map in merging list nested in merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "add field to map in merging list nested in merging list with value conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 1
        value: a
        other: c
      - name: 2
        value: b
        other: d
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 1
        value: 1
        other: c
      - name: 2
        value: 2
        other: d
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "add field to map in merging list nested in merging list with deletion conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 2
        value: 2
        other: d
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 1
      - name: 2
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 1
        value: 1
      - name: 2
        value: 2
        other: d
  - name: 2
    other: b
`),
		},
	},

	{
		Description: "add field to map in merging list nested in merging list with deletion conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 1
      - name: 2
        value: 2
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 2
      - name: 1
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    mergingList:
      - name: 2
        value: 2
      - name: 1
        value: 1
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 2
        value: 2
        other: d
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - $setElementOrder/mergingList:
      - name: 2
      - name: 1
    name: 1
    mergingList:
      - name: 1
        value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
    mergingList:
      - name: 2
        value: 2
        other: d
      - name: 1
        value: 1
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "add map to merging list by pointer",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergeItemPtr:
  - name: 1
`),
			TwoWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - name: 2
`),
			Modified: []byte(`
mergeItemPtr:
  - name: 1
  - name: 2
`),
			Current: []byte(`
mergeItemPtr:
  - name: 1
    other: a
  - name: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - name: 2
`),
			Result: []byte(`
mergeItemPtr:
  - name: 1
    other: a
  - name: 2
  - name: 3
`),
		},
	},
	{
		Description: "add map to merging list by pointer with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergeItemPtr:
  - name: 1
`),
			TwoWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - name: 2
`),
			Modified: []byte(`
mergeItemPtr:
  - name: 1
  - name: 2
`),
			Current: []byte(`
mergeItemPtr:
  - name: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergeItemPtr:
  - name: 1
  - name: 2
mergeItemPtr:
  - name: 1
  - name: 2
`),
			Result: []byte(`
mergeItemPtr:
  - name: 1
  - name: 2
  - name: 3
`),
		},
	},
	{
		Description: "add field to map in merging list by pointer",
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
        other: a
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
        other: a
      - name: 2
        value: 2
        other: b
  - name: 2
    other: b
`),
		},
	},
}
