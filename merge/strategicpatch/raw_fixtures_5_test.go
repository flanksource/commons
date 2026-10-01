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

var strategicMergePatchRawTestCases5 = []StrategicMergePatchRawTestCase{
	{
		Description: "retainKeys map changes a field in a nested map with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: old
    key2: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: new
    key2: b
    key3: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: modified
    key2: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key1: modified
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - simpleMap
    - value
  simpleMap:
    key1: modified
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  simpleMap:
    key1: modified
    key2: b
    key3: c
`),
		},
	},
	{
		Description: "retainKeys map replaces non-merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: c
  - name: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  nonMergingList:
  - name: a
  - name: c
  - name: b
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  nonMergingList:
  - name: a
  - name: c
  - name: b
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: c
  - name: b
`),
		},
	},
	{
		Description: "retainKeys map nested non-merging list with no change",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
		},
	},
	{
		Description: "retainKeys map nested non-merging list with no change with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
  - name: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - nonMergingList
    - value
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  nonMergingList:
  - name: a
  - name: b
`),
		},
	},
	{
		Description: "retainKeys map deletes nested non-merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
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
  nonMergingList: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
  nonMergingList: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map delete nested non-merging list with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  nonMergingList:
  - name: a
  - name: b
  - name: c
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
  nonMergingList: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
  nonMergingList: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map nested merging int list with no change",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  mergingIntList:
  - 1
  - 2
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
    - value
  $setElementOrder/mergingIntList:
    - 1
    - 2
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  mergingIntList:
  - 1
  - 2
  - 3
`),
		},
	},
	{
		Description: "retainKeys map adds an item in nested merging int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 4
`),
			TwoWay: []byte(`
retainKeysMap:
  $setElementOrder/mergingIntList:
    - 1
    - 2
    - 4
  $retainKeys:
    - mergingIntList
    - name
  mergingIntList:
  - 4
`),
			ThreeWay: []byte(`
retainKeysMap:
  $setElementOrder/mergingIntList:
    - 1
    - 2
    - 4
  $retainKeys:
    - mergingIntList
    - name
  mergingIntList:
  - 4
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 4
  - 3
`),
		},
	},
	{
		Description: "retainKeys map deletes an item in nested merging int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 3
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
  $deleteFromPrimitiveList/mergingIntList:
  - 2
  $setElementOrder/mergingIntList:
    - 1
    - 3
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
  $deleteFromPrimitiveList/mergingIntList:
  - 2
  $setElementOrder/mergingIntList:
    - 1
    - 3
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 3
  - 4
`),
		},
	},
	{
		Description: "retainKeys map adds an item and deletes an item in nested merging int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
  - 4
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 3
  - 5
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
  mergingIntList:
  - 5
  $deleteFromPrimitiveList/mergingIntList:
  - 2
  $setElementOrder/mergingIntList:
    - 1
    - 3
    - 5
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingIntList
    - name
  mergingIntList:
  - 5
  $deleteFromPrimitiveList/mergingIntList:
  - 2
  $setElementOrder/mergingIntList:
    - 1
    - 3
    - 5
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 3
  - 5
  - 4
`),
		},
	},
	{
		Description: "retainKeys map deletes nested merging int list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingIntList:
  - 1
  - 2
  - 3
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  mergingIntList: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
  mergingIntList: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
`),
		},
	},
	{
		Description: "retainKeys map nested merging list with no change",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
  - name: c
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  value: bar
  mergingList:
  - name: a
  - name: b
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
    - value
  value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
    - value
  $setElementOrder/mergingList:
    - name: a
    - name: b
  value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
  mergingList:
  - name: a
  - name: b
  - name: c
`),
		},
	},
	{
		Description: "retainKeys map adds an item in nested merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
  - name: x
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
  - name: c
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
    - name: b
    - name: c
  mergingList:
  - name: c
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
    - name: b
    - name: c
  mergingList:
  - name: c
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
  - name: c
  - name: x
`),
		},
	},
	{
		Description: "retainKeys map changes an item in nested merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
    value: foo
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
    value: foo
  - name: x
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
    value: bar
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
    - name: b
  mergingList:
  - name: b
    value: bar
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
    - name: b
  mergingList:
  - name: b
    value: bar
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
    value: bar
  - name: x
`),
		},
	},
	{
		Description: "retainKeys map deletes nested merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
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
  mergingList: null
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - name
    - value
  value: bar
  mergingList: null
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  value: bar
`),
		},
	},
	{
		Description: "retainKeys map deletes an item in nested merging list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
`),
			Current: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: b
  - name: x
`),
			Modified: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
`),
			TwoWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
  mergingList:
  - name: b
    $patch: delete
`),
			ThreeWay: []byte(`
retainKeysMap:
  $retainKeys:
    - mergingList
    - name
  $setElementOrder/mergingList:
    - name: a
  mergingList:
  - name: b
    $patch: delete
`),
			Result: []byte(`
retainKeysMap:
  name: foo
  mergingList:
  - name: a
  - name: x
`),
		},
	},
}
