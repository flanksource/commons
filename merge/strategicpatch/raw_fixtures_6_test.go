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

var strategicMergePatchRawTestCases6 = []StrategicMergePatchRawTestCase{
	{
		Description: "retainKeys list of maps clears a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			TwoWay: []byte(`{}`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
		},
	},
	{
		Description: "retainKeys list of maps clears a field with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: old
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: new
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: modified
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: modified
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: modified
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: modified
`),
		},
	},
	{
		Description: "retainKeys list of maps changes a field and clear a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: old
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: old
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: new
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: new
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: new
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: new
`),
		},
	},
	{
		Description: "retainKeys list of maps changes a field and clear a field with conflict",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: old
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: modified
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: new
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: new
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: new
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: new
`),
		},
	},
	{
		Description: "retainKeys list of maps adds a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: a
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: a
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
		},
	},
	{
		Description: "retainKeys list of maps adds a field and clear a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: a
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
    - value
  name: foo
  value: a
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
		},
	},
	{
		Description: "retainKeys list of maps deletes a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
  name: foo
  value: null
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
  name: foo
  value: null
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
		},
	},
	{
		Description: "retainKeys list of maps deletes a field and clear a field",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
`),
			Current: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
  value: a
  other: x
`),
			Modified: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
			TwoWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
  name: foo
  value: null
`),
			ThreeWay: []byte(`
$setElementOrder/retainKeysMergingList:
  - name: bar
  - name: foo
retainKeysMergingList:
- $retainKeys:
    - name
  name: foo
  value: null
`),
			Result: []byte(`
retainKeysMergingList:
- name: bar
- name: foo
`),
		},
	},
	{
		Description: "delete and reorder in one list, reorder in another",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
mergingList:
- name: a
  value: a
- name: b
  value: b
mergeItemPtr:
- name: c
  value: c
- name: d
  value: d
`),
			Current: []byte(`
mergingList:
- name: a
  value: a
- name: b
  value: b
mergeItemPtr:
- name: c
  value: c
- name: d
  value: d
`),
			Modified: []byte(`
mergingList:
- name: b
  value: b
mergeItemPtr:
- name: d
  value: d
- name: c
  value: c
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
- name: b
$setElementOrder/mergeItemPtr:
- name: d
- name: c
mergingList:
- $patch: delete
  name: a
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
- name: b
$setElementOrder/mergeItemPtr:
- name: d
- name: c
mergingList:
- $patch: delete
  name: a
`),
			Result: []byte(`
mergingList:
- name: b
  value: b
mergeItemPtr:
- name: d
  value: d
- name: c
  value: c
`),
		},
	},
}
