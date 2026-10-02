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
// strategicMergePatchRawTestCases is split across raw_fixtures_{1..6}_test.go to keep files small.

package strategicpatch

var strategicMergePatchRawTestCases1 = []StrategicMergePatchRawTestCase{
	{
		Description: "nested patch merge with empty list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
name: hi
`),
			Current: []byte(`
name: hi
mergingList:
- name: hello2
`),
			Modified: []byte(`
name: hi
mergingList:
- name: hello
- $patch: delete
  name: doesntexist
`),
			TwoWay: []byte(`
mergingList:
- name: hello
- $patch: delete
  name: doesntexist
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
- name: hello
- name: doesntexist
mergingList:
- name: hello
`),
			TwoWayResult: []byte(`
name: hi
mergingList:
- name: hello
`),
			Result: []byte(`
name: hi
mergingList:
- name: hello
- name: hello2
`),
		},
	},
	{
		Description: "delete items in lists of scalars",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 2
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 1
  - 2
  - 4
`),
		},
	},
	{
		Description: "delete all duplicate items in lists of scalars",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 2
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 3
  - 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 1
  - 2
  - 4
`),
		},
	},
	{
		Description: "add and delete items in lists of scalars",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
  - 4
$deleteFromPrimitiveList/mergingIntList:
  - 3
mergingIntList:
  - 4
`),
			Modified: []byte(`
mergingIntList:
  - 1
  - 2
  - 4
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 1
  - 2
  - 4
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 1
  - 2
  - 4
`),
		},
	},
	{
		Description: "merge lists of maps",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
    value: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 4
  - name: 1
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
  - name: 4
    value: 4
  - name: 1
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
  - name: 4
  - name: 1
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
  - name: 4
    value: 4
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
		},
	},
	{
		Description: "merge lists of maps with conflict",
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
  - name: 2
  - name: 3
mergingList:
  - name: 3
    value: 3
`),
			Modified: []byte(`
mergingList:
  - name: 1
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
    value: 3
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
  - name: 3
mergingList:
  - name: 2
    value: 2
  - name: 3
    value: 3
`),
			Result: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 2
    value: 2
    other: b
  - name: 3
    value: 3
`),
		},
	},
	{
		Description: "add field to map in merging list",
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
  - name: 2
mergingList:
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: 1
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
    value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "add field to map in merging list",
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
  - name: 2
mergingList:
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: 1
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
    value: 1
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "add field to map in merging list with conflict",
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
  - name: 2
mergingList:
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    other: a
  - name: 3
    value: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
  - name: 3
    value: 2
    other: b
`),
		},
	},
	{
		Description: "add duplicate field to map in merging list",
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
  - name: 2
mergingList:
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: 1
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
			ThreeWay: []byte(`{}`),
			Result: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "add an item that already exists in current object in merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
    value: a
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
  - name: 3
mergingList:
  - name: 3
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: a
  - name: 2
  - name: 3
`),
			Current: []byte(`
mergingList:
  - name: 1
    value: a
    other: x
  - name: 2
  - name: 3
`),
			ThreeWay: []byte(`{}`),
			Result: []byte(`
mergingList:
  - name: 1
    value: a
    other: x
  - name: 2
  - name: 3
`),
		},
	},
	{
		Description: "add duplicate field to map in merging list with conflict",
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
  - name: 2
mergingList:
  - name: 1
    value: 1
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: 1
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 3
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 2
    value: 2
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: 1
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "replace map field value in merging list",
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
    value: a
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: a
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
    value: a
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: a
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "replace map field value in merging list with conflict",
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
    value: a
`),
			Modified: []byte(`
mergingList:
  - name: 1
    value: a
  - name: 2
    value: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
    value: 3
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
    value: a
`),
			Result: []byte(`
mergingList:
  - name: 1
    value: a
    other: a
  - name: 2
    value: 2
    other: b
`),
		},
	},
	{
		Description: "delete map from merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 1
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "delete map from merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
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
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "delete missing map from merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 2
    other: b
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
    other: b
`),
		},
	},
	{
		Description: "delete missing map from merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
  - name: 1
  - name: 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 1
    $patch: delete
`),
			Modified: []byte(`
mergingList:
  - name: 2
`),
			Current: []byte(`
mergingList:
  - name: 3
    other: a
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
mergingList:
  - name: 2
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
  - name: 3
    other: a
`),
		},
	},
	{
		Description: "add map and delete map from merging list",
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
  - name: 2
    other: b
  - name: 4
    other: c
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 2
  - name: 3
mergingList:
  - name: 3
  - name: 1
    $patch: delete
`),
			Result: []byte(`
mergingList:
  - name: 2
    other: b
  - name: 4
    other: c
  - name: 3
`),
		},
	},
}
