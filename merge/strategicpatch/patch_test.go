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
//
// Upstream runs every case against three schemas (Go struct tags, OpenAPI v2 and
// OpenAPI v3) and asserts both the generated patch and its application. This
// port keeps the struct-tag schema only and asserts the application of each
// fixture's expected patch — upstream asserts that patch equals the generated
// one before applying it, so the applied input is the same.

package strategicpatch

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/flanksource/commons/merge/strategicpatch/internal/json"
	"github.com/flanksource/commons/merge/strategicpatch/mergepatch"
)

// rawExtension stands in for k8s.io/apimachinery/pkg/runtime.RawExtension on
// MergeItem.ReplacingItem. That field carries patchStrategy "replace", so the
// struct schema never recurses into its type and only its tags matter.
type rawExtension struct {
	Raw []byte
}

// strategicMergePatchRawTestCases is upstream's single fixture list, split
// across raw_fixtures_{1..6}_test.go.
var strategicMergePatchRawTestCases = slices.Concat(
	strategicMergePatchRawTestCases1,
	strategicMergePatchRawTestCases2,
	strategicMergePatchRawTestCases3,
	strategicMergePatchRawTestCases4,
	strategicMergePatchRawTestCases5,
	strategicMergePatchRawTestCases6,
)

type SortMergeListTestCases struct {
	TestCases []SortMergeListTestCase
}

type SortMergeListTestCase struct {
	Description string
	Original    map[string]interface{}
	Sorted      map[string]interface{}
}

type StrategicMergePatchTestCases struct {
	TestCases []StrategicMergePatchTestCase
}

type StrategicMergePatchTestCase struct {
	Description string
	StrategicMergePatchTestCaseData
}

type StrategicMergePatchRawTestCase struct {
	Description string
	StrategicMergePatchRawTestCaseData
}

type StrategicMergePatchTestCaseData struct {
	// Original is the original object (last-applied config in annotation)
	Original map[string]interface{}
	// Modified is the modified object (new config we want)
	Modified map[string]interface{}
	// Current is the current object (live config in the server)
	Current map[string]interface{}
	// TwoWay is the expected two-way merge patch diff between original and modified
	TwoWay map[string]interface{}
	// ThreeWay is the expected three-way merge patch
	ThreeWay map[string]interface{}
	// Result is the expected object after applying the three-way patch on current object.
	Result map[string]interface{}
	// TwoWayResult is the expected object after applying the two-way patch on current object.
	// If nil, Modified is used.
	TwoWayResult map[string]interface{}
}

// The meaning of each field is the same as StrategicMergePatchTestCaseData's.
// The difference is that all the fields in StrategicMergePatchRawTestCaseData are json-encoded data.
type StrategicMergePatchRawTestCaseData struct {
	Original      []byte
	Modified      []byte
	Current       []byte
	TwoWay        []byte
	ThreeWay      []byte
	Result        []byte
	TwoWayResult  []byte
	ExpectedError string
}

type MergeItem struct {
	Name                  string               `json:"name,omitempty"`
	Value                 string               `json:"value,omitempty"`
	Other                 string               `json:"other,omitempty"`
	MergingList           []MergeItem          `json:"mergingList,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	NonMergingList        []MergeItem          `json:"nonMergingList,omitempty"`
	MergingIntList        []int                `json:"mergingIntList,omitempty" patchStrategy:"merge"`
	NonMergingIntList     []int                `json:"nonMergingIntList,omitempty"`
	MergeItemPtr          *MergeItem           `json:"mergeItemPtr,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	SimpleMap             map[string]string    `json:"simpleMap,omitempty"`
	ReplacingItem         rawExtension         `json:"replacingItem,omitempty" patchStrategy:"replace"`
	JSONItem              struct{ Raw []byte } `json:"jsonItem,omitempty"`
	RetainKeysMap         RetainKeysMergeItem  `json:"retainKeysMap,omitempty" patchStrategy:"retainKeys"`
	RetainKeysMergingList []MergeItem          `json:"retainKeysMergingList,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

type RetainKeysMergeItem struct {
	Name           string            `json:"name,omitempty"`
	Value          string            `json:"value,omitempty"`
	Other          string            `json:"other,omitempty"`
	SimpleMap      map[string]string `json:"simpleMap,omitempty"`
	MergingIntList []int             `json:"mergingIntList,omitempty" patchStrategy:"merge"`
	MergingList    []MergeItem       `json:"mergingList,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	NonMergingList []MergeItem       `json:"nonMergingList,omitempty"`
}

var (
	mergeItem             MergeItem
	mergeItemStructSchema = PatchMetaFromStruct{T: GetTagStructTypeOrDie(mergeItem)}
)

func TestSortMergeLists(t *testing.T) {
	schemas := []LookupPatchMeta{
		mergeItemStructSchema,
	}

	tc := SortMergeListTestCases{}
	err := yaml.Unmarshal(sortMergeListTestCaseData, &tc)
	if err != nil {
		t.Errorf("can't unmarshal test cases: %s\n", err)
		return
	}

	for _, schema := range schemas {
		for _, c := range tc.TestCases {
			temp := testObjectToJSONOrFail(t, c.Original)
			got := sortJsonOrFail(t, temp, c.Description, schema)
			expected := testObjectToJSONOrFail(t, c.Sorted)
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("using %s error in test case: %s\ncannot sort object:\n%s\nexpected:\n%s\ngot:\n%s\n",
					getSchemaType(schema), c.Description, jsonToYAMLOrError(temp), jsonToYAMLOrError(expected), jsonToYAMLOrError(got))
			}
		}
	}
}

func TestCustomStrategicMergePatch(t *testing.T) {
	schemas := []LookupPatchMeta{
		mergeItemStructSchema,
	}

	tc := StrategicMergePatchTestCases{}
	err := yaml.Unmarshal(customStrategicMergePatchTestCaseData, &tc)
	if err != nil {
		t.Errorf("can't unmarshal test cases: %v\n", err)
		return
	}

	for _, c := range tc.TestCases {
		t.Run(c.Description, func(t *testing.T) {
			for _, schema := range schemas {
				t.Run(schema.Name(), func(t *testing.T) {
					original, expectedTwoWayPatch, _, expectedResult := twoWayTestCaseToJSONOrFail(t, c, schema)
					testPatchApplication(t, original, expectedTwoWayPatch, expectedResult, c.Description, "", schema)
				})

				for _, c := range customStrategicMergePatchRawTestCases {
					original, expectedTwoWayPatch, _, expectedResult := twoWayRawTestCaseToJSONOrFail(t, c)
					testPatchApplication(t, original, expectedTwoWayPatch, expectedResult, c.Description, c.ExpectedError, schema)
				}
			}
		})
	}
}

func TestStrategicMergePatch(t *testing.T) {
	testStrategicMergePatchWithCustomArgumentsUsingStruct(t, "bad struct",
		"{}", "{}", []byte("<THIS IS NOT A STRUCT>"), mergepatch.ErrBadArgKind(struct{}{}, []byte{}))

	schemas := []LookupPatchMeta{
		mergeItemStructSchema,
	}

	tc := StrategicMergePatchTestCases{}
	err := yaml.Unmarshal(createStrategicMergePatchTestCaseData, &tc)
	if err != nil {
		t.Errorf("can't unmarshal test cases: %s\n", err)
		return
	}

	for _, schema := range schemas {
		t.Run(schema.Name(), func(t *testing.T) {

			testStrategicMergePatchWithCustomArguments(t, "bad original",
				"<THIS IS NOT JSON>", "{}", schema, mergepatch.ErrBadJSONDoc)
			testStrategicMergePatchWithCustomArguments(t, "bad patch",
				"{}", "<THIS IS NOT JSON>", schema, mergepatch.ErrBadJSONDoc)
			testStrategicMergePatchWithCustomArguments(t, "nil struct",
				"{}", "{}", nil, mergepatch.ErrBadArgKind(struct{}{}, nil))

			for _, c := range tc.TestCases {
				t.Run(c.Description+"/TwoWay", func(t *testing.T) {
					testTwoWayPatch(t, c, schema)
				})
				t.Run(c.Description+"/ThreeWay", func(t *testing.T) {
					testThreeWayPatch(t, c, schema)
				})
			}
		})

		// run multiple times to exercise different map traversal orders
		for i := 0; i < 10; i++ {
			for _, c := range strategicMergePatchRawTestCases {
				t.Run(c.Description+"/TwoWay", func(t *testing.T) {
					testTwoWayPatchForRawTestCase(t, c, schema)
				})
				t.Run(c.Description+"/ThreeWay", func(t *testing.T) {
					testThreeWayPatchForRawTestCase(t, c, schema)
				})
			}
		}
	}
}

func testStrategicMergePatchWithCustomArgumentsUsingStruct(t *testing.T, description, original, patch string, dataStruct interface{}, expected error) {
	schema, actual := NewPatchMetaFromStruct(dataStruct)
	// If actual is not nil, check error. If errors match, return.
	if actual != nil {
		checkErrorsEqual(t, description, expected, actual, schema)
		return
	}
	testStrategicMergePatchWithCustomArguments(t, description, original, patch, schema, expected)
}

func testStrategicMergePatchWithCustomArguments(t *testing.T, description, original, patch string, schema LookupPatchMeta, expected error) {
	_, actual := StrategicMergePatch([]byte(original), []byte(patch), schema)
	checkErrorsEqual(t, description, expected, actual, schema)
}

func checkErrorsEqual(t *testing.T, description string, expected, actual error, schema LookupPatchMeta) {
	if actual != expected {
		if actual == nil {
			t.Errorf("using %s expected error: %s\ndid not occur in test case: %s", getSchemaType(schema), expected, description)
			return
		}

		if expected == nil || actual.Error() != expected.Error() {
			t.Errorf("using %s unexpected error: %s\noccurred in test case: %s", getSchemaType(schema), actual, description)
			return
		}
	}
}

// testTwoWayPatch is the apply half of upstream's testTwoWayPatch: it applies
// the fixture's two-way patch to the original.
func testTwoWayPatch(t *testing.T, c StrategicMergePatchTestCase, schema LookupPatchMeta) {
	original, expectedPatch, _, expectedResult := twoWayTestCaseToJSONOrFail(t, c, schema)
	testPatchApplication(t, original, expectedPatch, expectedResult, c.Description, "", schema)
}

func testTwoWayPatchForRawTestCase(t *testing.T, c StrategicMergePatchRawTestCase, schema LookupPatchMeta) {
	original, expectedPatch, _, expectedResult := twoWayRawTestCaseToJSONOrFail(t, c)
	testPatchApplication(t, original, expectedPatch, expectedResult, c.Description, c.ExpectedError, schema)
}

func twoWayTestCaseToJSONOrFail(t *testing.T, c StrategicMergePatchTestCase, schema LookupPatchMeta) ([]byte, []byte, []byte, []byte) {
	expectedResult := c.TwoWayResult
	if expectedResult == nil {
		expectedResult = c.Modified
	}
	return sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Original), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.TwoWay), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Modified), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, expectedResult), c.Description, schema)
}

func twoWayRawTestCaseToJSONOrFail(t *testing.T, c StrategicMergePatchRawTestCase) ([]byte, []byte, []byte, []byte) {
	expectedResult := c.TwoWayResult
	if expectedResult == nil {
		expectedResult = c.Modified
	}
	return yamlToJSONOrError(t, c.Original),
		yamlToJSONOrError(t, c.TwoWay),
		yamlToJSONOrError(t, c.Modified),
		yamlToJSONOrError(t, expectedResult)
}

// testThreeWayPatch is the apply half of upstream's testThreeWayPatch: it
// applies the fixture's three-way patch to the current object. Upstream only
// applies a three-way patch when the case has a result; a conflict case
// without one stops at patch creation, which this port does not have.
func testThreeWayPatch(t *testing.T, c StrategicMergePatchTestCase, schema LookupPatchMeta) {
	if len(c.Result) < 1 {
		return
	}
	_, _, current, expected, result := threeWayTestCaseToJSONOrFail(t, c, schema)
	testPatchApplication(t, current, expected, result, c.Description, "", schema)
}

func testThreeWayPatchForRawTestCase(t *testing.T, c StrategicMergePatchRawTestCase, schema LookupPatchMeta) {
	if len(c.Result) < 1 {
		return
	}
	_, _, current, expected, result := threeWayRawTestCaseToJSONOrFail(t, c)
	testPatchApplication(t, current, expected, result, c.Description, c.ExpectedError, schema)
}

func threeWayTestCaseToJSONOrFail(t *testing.T, c StrategicMergePatchTestCase, schema LookupPatchMeta) ([]byte, []byte, []byte, []byte, []byte) {
	return sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Original), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Modified), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Current), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.ThreeWay), c.Description, schema),
		sortJsonOrFail(t, testObjectToJSONOrFail(t, c.Result), c.Description, schema)
}

func threeWayRawTestCaseToJSONOrFail(t *testing.T, c StrategicMergePatchRawTestCase) ([]byte, []byte, []byte, []byte, []byte) {
	return yamlToJSONOrError(t, c.Original),
		yamlToJSONOrError(t, c.Modified),
		yamlToJSONOrError(t, c.Current),
		yamlToJSONOrError(t, c.ThreeWay),
		yamlToJSONOrError(t, c.Result)
}

func testPatchApplication(t *testing.T, original, patch, expected []byte, description, expectedError string, schema LookupPatchMeta) {
	result, err := StrategicMergePatchUsingLookupPatchMeta(original, patch, schema)
	if len(expectedError) != 0 {
		if err != nil && strings.Contains(err.Error(), expectedError) {
			return
		}
		t.Errorf("using %s expected error should contain:\n%s\nin test case: %s\nbut got:\n%s\n", getSchemaType(schema), expectedError, description, err)
	}
	if err != nil {
		t.Errorf("using %s error: %s\nin test case: %s\ncannot apply patch:\n%s\nto original:\n%s\n",
			getSchemaType(schema), err, description, jsonToYAMLOrError(patch), jsonToYAMLOrError(original))
		return
	}

	if !reflect.DeepEqual(result, expected) {
		format := "using error in test case: %s\npatch application failed:\noriginal:\n%s\npatch:\n%s\nexpected:\n%s\ngot:\n%s\n"
		t.Errorf(format, description,
			jsonToYAMLOrError(original), jsonToYAMLOrError(patch),
			jsonToYAMLOrError(expected), jsonToYAMLOrError(result))
		return
	}
}

func testObjectToJSONOrFail(t *testing.T, o map[string]interface{}) []byte {
	if o == nil {
		return nil
	}

	j, err := toJSON(o)
	if err != nil {
		t.Error(err)
	}
	return j
}

func sortJsonOrFail(t *testing.T, j []byte, description string, schema LookupPatchMeta) []byte {
	if j == nil {
		return nil
	}
	r, err := sortMergeListsByName(j, schema)
	if err != nil {
		t.Errorf("using %s error: %s\n in test case: %s\ncannot sort object:\n%s\n", getSchemaType(schema), err, description, j)
		return nil
	}

	return r
}

func getSchemaType(schema LookupPatchMeta) string {
	return reflect.TypeOf(schema).String()
}

func jsonToYAMLOrError(j []byte) string {
	y, err := jsonToYAML(j)
	if err != nil {
		return err.Error()
	}

	return string(y)
}

func toJSON(v interface{}) ([]byte, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed: %v\n%#v\n", err, v)
	}

	return j, nil
}

func jsonToYAML(j []byte) ([]byte, error) {
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		return nil, fmt.Errorf("json to yaml failed: %v\n%v\n", err, j)
	}

	return y, nil
}

func yamlToJSON(y []byte) ([]byte, error) {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, fmt.Errorf("yaml to json failed: %v\n%v\n", err, y)
	}

	return j, nil
}

func yamlToJSONOrError(t *testing.T, y []byte) []byte {
	j, err := yamlToJSON(y)
	if err != nil {
		t.Errorf("%v", err)
	}

	return j
}
