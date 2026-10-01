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

var strategicMergePatchRawTestCases4 = []StrategicMergePatchRawTestCase{
	{
		Description: "delete an item in a int list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
  - 2
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
		Description: "add and delete an item in a int list and preserve order",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 4
  - 2
  - 1
mergingIntList:
  - 4
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 4
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
  - 4
  - 2
  - 1
mergingIntList:
  - 4
$deleteFromPrimitiveList/mergingIntList:
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 4
  - 2
  - 1
`),
		},
	},
	{
		Description: "set elements order in a int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
  - 4
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Modified: []byte(`
mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 3
  - 4
  - 2
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Result: []byte(`
mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
		},
	},
	{
		Description: "set elements order in a int list with server-only items",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 3
  - 4
  - 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Modified: []byte(`
mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 3
  - 4
  - 2
  - 9
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 4
  - 2
  - 3
  - 1
`),
			Result: []byte(`
mergingIntList:
  - 4
  - 2
  - 3
  - 1
  - 9
`),
		},
	},
	{
		Description: "set elements order in a int list with server-only items 2",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 9
  - 3
  - 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 2
  - 1
  - 9
  - 4
  - 3
`),
		},
	},
	{
		Description: "set elements order in a int list with server-only items 3",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Modified: []byte(`
mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Current: []byte(`
mergingIntList:
  - 1
  - 2
  - 7
  - 9
  - 8
  - 3
  - 4
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
  - 2
  - 1
  - 4
  - 3
`),
			Result: []byte(`
mergingIntList:
  - 2
  - 1
  - 7
  - 9
  - 8
  - 4
  - 3
`),
		},
	},
	{
		// This test case is used just to demonstrate the behavior when dealing with a list with duplicate
		Description: "behavior of set element order for a merging list with duplicate",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
- name: 1
- name: 2
  value: dup1
- name: 3
- name: 2
  value: dup2
- name: 4
`),
			Current: []byte(`
mergingList:
- name: 1
- name: 2
  value: dup1
- name: 3
- name: 2
  value: dup2
- name: 4
`),
			Modified: []byte(`
mergingList:
- name: 2
  value: dup1
- name: 1
- name: 4
- name: 3
- name: 2
  value: dup2
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
- name: 2
- name: 1
- name: 4
- name: 3
- name: 2
`),
			TwoWayResult: []byte(`
mergingList:
- name: 2
  value: dup1
- name: 2
  value: dup2
- name: 1
- name: 4
- name: 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
- name: 2
- name: 1
- name: 4
- name: 3
- name: 2
`),
			Result: []byte(`
mergingList:
- name: 2
  value: dup1
- name: 2
  value: dup2
- name: 1
- name: 4
- name: 3
`),
		},
	},
	{
		// This test case is used just to demonstrate the behavior when dealing with a list with duplicate
		Description: "behavior of set element order for a merging int list with duplicate",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingIntList:
- 1
- 2
- 3
- 2
- 4
`),
			Current: []byte(`
mergingIntList:
- 1
- 2
- 3
- 2
- 4
`),
			Modified: []byte(`
mergingIntList:
- 2
- 1
- 4
- 3
- 2
`),
			TwoWay: []byte(`
$setElementOrder/mergingIntList:
- 2
- 1
- 4
- 3
- 2
`),
			TwoWayResult: []byte(`
mergingIntList:
- 2
- 2
- 1
- 4
- 3
`),
			ThreeWay: []byte(`
$setElementOrder/mergingIntList:
- 2
- 1
- 4
- 3
- 2
`),
			Result: []byte(`
mergingIntList:
- 2
- 2
- 1
- 4
- 3
`),
		},
	},
	{
		Description: "retainKeys map should clear defaulted field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`{}`),
			Current: []byte(`
retainKeysMap:
  value: foo
`),
			Modified: []byte(`
retainKeysMap:
  other: bar
`),
			TwoWay: []byte(`
retainKeysMap:
  other: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - other
  other: bar
`),
			Result: []byte(`
retainKeysMap:
  other: bar
`),
		},
	},
	{
		Description: "retainKeys map should clear defaulted field with conflict (discriminated union)",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`{}`),
			Current: []byte(`
retainKeysMap:
  name: type1
  value: foo
`),
			Modified: []byte(`
retainKeysMap:
  name: type2
  other: bar
`),
			TwoWay: []byte(`
retainKeysMap:
  name: type2
  other: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - other
  name: type2
  other: bar
`),
			Result: []byte(`
retainKeysMap:
  name: type2
  other: bar
`),
		},
	},
	{
		Description: "retainKeys map adds a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
`),
			Current: []byte(`
retainKeysMap:
  name: foo
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map adds a field and clear a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  other: a
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map deletes a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  value: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  value: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
`),
		},
	},
	{
		Description: "retainKeys map deletes a field and clears a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  other: a
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  value: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  value: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
`),
		},
	},
	{
		Description: "retainKeys map clears a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  other: a
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
			TwoWay: []byte(`{}`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map nested map with no change",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  simpleMap:
    key1: a
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  simpleMap:
    key1: a
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
`),
		},
	},
	{
		Description: "retainKeys map adds a field in a nested map",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key3: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key2: b
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key2: b
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
    key3: c
`),
		},
	},
	{
		Description: "retainKeys map deletes a field in a nested map",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
    key3: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key2: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key2: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key3: c
`),
		},
	},
	{
		Description: "retainKeys map changes a field in a nested map",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: a
    key2: b
    key3: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: x
    key2: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key1: x
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key1: x
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: x
    key2: b
    key3: c
`),
		},
	},
}
