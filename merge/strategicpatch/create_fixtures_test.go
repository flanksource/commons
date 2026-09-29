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

// These are test cases for StrategicMergePatch, to assert that applying  a patch
// yields the correct outcome. They are also test cases for CreateTwoWayMergePatch
// and CreateThreeWayMergePatch, to assert that they both generate the correct patch
// for the given set of input documents.
var createStrategicMergePatchTestCaseData = []byte(`
testCases:
  - description: nil original
    twoWay:
      name: 1
      value: 1
    modified:
      name: 1
      value: 1
    current:
      name: 1
      other: a
    threeWay:
      value: 1
    result:
      name: 1
      value: 1
      other: a
  - description: nil patch
    original:
      name: 1
    twoWay:
      {}
    modified:
      name: 1
    current:
      name: 1
    threeWay:
      {}
    result:
      name: 1
  - description: add field to map
    original:
      name: 1
    twoWay:
      value: 1
    modified:
      name: 1
      value: 1
    current:
      name: 1
      other: a
    threeWay:
      value: 1
    result:
      name: 1
      value: 1
      other: a
  - description: add field to map with conflict
    original:
      name: 1
    twoWay:
      value: 1
    modified:
      name: 1
      value: 1
    current:
      name: a
      other: a
    threeWay:
      name: 1
      value: 1
    result:
      name: 1
      value: 1
      other: a
  - description: add field and delete field from map
    original:
      name: 1
    twoWay:
      name: null
      value: 1
    modified:
      value: 1
    current:
      name: 1
      other: a
    threeWay:
      name: null
      value: 1
    result:
      value: 1
      other: a
  - description: add field and delete field from map with conflict
    original:
      name: 1
    twoWay:
      name: null
      value: 1
    modified:
      value: 1
    current:
      name: a
      other: a
    threeWay:
      name: null
      value: 1
    result:
      value: 1
      other: a
  - description: delete field from nested map
    original:
      simpleMap:
        key1: 1
        key2: 1
    twoWay:
      simpleMap:
        key2: null
    modified:
      simpleMap:
        key1: 1
    current:
      simpleMap:
        key1: 1
        key2: 1
        other: a
    threeWay:
      simpleMap:
        key2: null
    result:
      simpleMap:
        key1: 1
        other: a
  - description: delete field from nested map with conflict
    original:
      simpleMap:
        key1: 1
        key2: 1
    twoWay:
      simpleMap:
        key2: null
    modified:
      simpleMap:
        key1: 1
    current:
      simpleMap:
        key1: a
        key2: 1
        other: a
    threeWay:
      simpleMap:
        key1: 1
        key2: null
    result:
      simpleMap:
        key1: 1
        other: a
  - description: delete all fields from map
    original:
      name: 1
      value: 1
    twoWay:
      name: null
      value: null
    modified: {}
    current:
      name: 1
      value: 1
      other: a
    threeWay:
      name: null
      value: null
    result:
      other: a
  - description: delete all fields from map with conflict
    original:
      name: 1
      value: 1
    twoWay:
      name: null
      value: null
    modified: {}
    current:
      name: 1
      value: a
      other: a
    threeWay:
      name: null
      value: null
    result:
      other: a
  - description: add field and delete all fields from map
    original:
      name: 1
      value: 1
    twoWay:
      name: null
      value: null
      other: a
    modified:
      other: a
    current:
      name: 1
      value: 1
      other: a
    threeWay:
      name: null
      value: null
    result:
      other: a
  - description: add field and delete all fields from map with conflict
    original:
      name: 1
      value: 1
    twoWay:
      name: null
      value: null
      other: a
    modified:
      other: a
    current:
      name: 1
      value: 1
      other: b
    threeWay:
      name: null
      value: null
      other: a
    result:
      other: a
  - description: replace list of scalars
    original:
      nonMergingIntList:
        - 1
        - 2
    twoWay:
      nonMergingIntList:
        - 2
        - 3
    modified:
      nonMergingIntList:
        - 2
        - 3
    current:
      nonMergingIntList:
        - 1
        - 2
    threeWay:
      nonMergingIntList:
        - 2
        - 3
    result:
      nonMergingIntList:
        - 2
        - 3
  - description: replace list of scalars with conflict
    original:
      nonMergingIntList:
        - 1
        - 2
    twoWay:
      nonMergingIntList:
        - 2
        - 3
    modified:
      nonMergingIntList:
        - 2
        - 3
    current:
      nonMergingIntList:
        - 1
        - 4
    threeWay:
      nonMergingIntList:
        - 2
        - 3
    result:
      nonMergingIntList:
        - 2
        - 3
  - description: delete all maps from merging list
    original:
      mergingList:
        - name: 1
        - name: 2
    twoWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    modified:
      mergingList: []
    current:
      mergingList:
        - name: 1
        - name: 2
    threeWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    result:
      mergingList: []
  - description: delete all maps from merging list with conflict
    original:
      mergingList:
        - name: 1
        - name: 2
    twoWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    modified:
      mergingList: []
    current:
      mergingList:
        - name: 1
          other: a
        - name: 2
          other: b
    threeWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    result:
      mergingList: []
  - description: delete all maps from empty merging list
    original:
      mergingList:
        - name: 1
        - name: 2
    twoWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    modified:
      mergingList: []
    current:
      mergingList: []
    threeWay:
      mergingList:
        - name: 1
          $patch: delete
        - name: 2
          $patch: delete
    result:
      mergingList: []
  - description: merge empty merging lists
    original:
      mergingList: []
    twoWay:
      {}
    modified:
      mergingList: []
    current:
      mergingList: []
    threeWay:
      {}
    result:
      mergingList: []
  - description: defined null values should propagate overwrite current fields (with conflict)
    original:
      name: 2
    twoWay:
      name: 1
      value: 1
      other: null
    twoWayResult:
      name: 1
      value: 1
    modified:
      name: 1
      value: 1
      other: null
    current:
      name: a
      other: a
    threeWay:
      name: 1
      value: 1
      other: null
    result:
      name: 1
      value: 1
  - description: defined null values should propagate removing original fields
    original:
      name: original-name
      value: original-value
    current:
      name: original-name
      value: original-value
      other: current-other
    modified:
      name: modified-name
      value: null
    twoWay:
      name: modified-name
      value: null
    twoWayResult:
      name: modified-name
    threeWay:
      name: modified-name
      value: null
    result:
      name: modified-name
      other: current-other
  - description: nil patch with retainKeys map
    original:
      name: a
      retainKeysMap:
        name: foo
    current:
      name: a
      value: b
      retainKeysMap:
        name: foo
    modified:
      name: a
      retainKeysMap:
        name: foo
    twoWay: {}
    threeWay: {}
    result:
      name: a
      value: b
      retainKeysMap:
        name: foo
  - description: retainKeys map with no change should not be present
    original:
      name: a
      retainKeysMap:
        name: foo
    current:
      name: a
      other: c
      retainKeysMap:
        name: foo
    modified:
      name: a
      value: b
      retainKeysMap:
        name: foo
    twoWay:
      value: b
    threeWay:
      value: b
    result:
      name: a
      value: b
      other: c
      retainKeysMap:
        name: foo
`)
