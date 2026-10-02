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

import (
	"sort"
	"testing"
)

type PrecisionItem struct {
	Name    string  `json:"name,omitempty"`
	Int32   int32   `json:"int32,omitempty"`
	Int64   int64   `json:"int64,omitempty"`
	Float32 float32 `json:"float32,omitempty"`
	Float64 float64 `json:"float64,omitempty"`
}

var (
	precisionItem             PrecisionItem
	precisionItemStructSchema = PatchMetaFromStruct{T: GetTagStructTypeOrDie(precisionItem)}
)

func TestNumberConversion(t *testing.T) {
	testcases := map[string]struct {
		Old            string
		New            string
		ExpectedPatch  string
		ExpectedResult string
	}{
		"empty": {
			Old:            `{}`,
			New:            `{}`,
			ExpectedPatch:  `{}`,
			ExpectedResult: `{}`,
		},
		"int32 medium": {
			Old:            `{"int32":1000000}`,
			New:            `{"int32":1000000,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"int32":1000000,"name":"newname"}`,
		},
		"int32 max": {
			Old:            `{"int32":2147483647}`,
			New:            `{"int32":2147483647,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"int32":2147483647,"name":"newname"}`,
		},
		"int64 medium": {
			Old:            `{"int64":1000000}`,
			New:            `{"int64":1000000,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"int64":1000000,"name":"newname"}`,
		},
		"int64 max": {
			Old:            `{"int64":9223372036854775807}`,
			New:            `{"int64":9223372036854775807,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"int64":9223372036854775807,"name":"newname"}`,
		},
		"float32 max": {
			Old:            `{"float32":3.4028234663852886e+38}`,
			New:            `{"float32":3.4028234663852886e+38,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"float32":3.4028234663852886e+38,"name":"newname"}`,
		},
		"float64 max": {
			Old:            `{"float64":1.7976931348623157e+308}`,
			New:            `{"float64":1.7976931348623157e+308,"name":"newname"}`,
			ExpectedPatch:  `{"name":"newname"}`,
			ExpectedResult: `{"float64":1.7976931348623157e+308,"name":"newname"}`,
		},
	}

	precisionItemSchemas := []LookupPatchMeta{
		precisionItemStructSchema,
	}

	for _, schema := range precisionItemSchemas {
		for k, tc := range testcases {
			// Apply half: the expected patch stands in for the generated one.
			patch := []byte(tc.ExpectedPatch)

			result, err := StrategicMergePatchUsingLookupPatchMeta([]byte(tc.Old), patch, schema)
			if err != nil {
				t.Errorf("using %s in testcase %s: unexpected error %v", getSchemaType(schema), k, err)
				continue
			}
			if tc.ExpectedResult != string(result) {
				t.Errorf("using %s in testcase %s: expected %s, got %s", getSchemaType(schema), k, tc.ExpectedResult, string(result))
				continue
			}
		}
	}
}

var replaceRawExtensionPatchTestCases = []StrategicMergePatchRawTestCase{
	{
		Description: "replace RawExtension field, rest unchanched",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
name: my-object
value: some-value
other: current-other
replacingItem:
  Some: Generic
  Yaml: Inside
  The: RawExtension
  Field: Period
`),
			Current: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Some: Generic
  Yaml: Inside
  The: RawExtension
  Field: Period
`),
			Modified: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			TwoWay: []byte(`
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			TwoWayResult: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			ThreeWay: []byte(`
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			Result: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
		},
	},
	{
		Description: "replace RawExtension field and merge list",
		StrategicMergePatchRawTestCaseData: StrategicMergePatchRawTestCaseData{
			Original: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
replacingItem:
  Some: Generic
  Yaml: Inside
  The: RawExtension
  Field: Period
`),
			Current: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 3
replacingItem:
  Some: Generic
  Yaml: Inside
  The: RawExtension
  Field: Period
`),
			Modified: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			TwoWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 2
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			TwoWayResult: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			ThreeWay: []byte(`
$setElementOrder/mergingList:
  - name: 1
  - name: 2
mergingList:
  - name: 2
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
			Result: []byte(`
name: my-object
value: some-value
other: current-other
mergingList:
  - name: 1
  - name: 2
  - name: 3
replacingItem:
  Newly: Modified
  Yaml: Inside
  The: RawExtension
`),
		},
	},
}

func TestReplaceWithRawExtension(t *testing.T) {
	schemas := []LookupPatchMeta{
		mergeItemStructSchema,
	}

	for _, schema := range schemas {
		for _, c := range replaceRawExtensionPatchTestCases {
			testTwoWayPatchForRawTestCase(t, c, schema)
			testThreeWayPatchForRawTestCase(t, c, schema)
		}
	}
}

func TestUnknownField(t *testing.T) {
	testcases := map[string]struct {
		Original string
		Current  string
		Modified string

		ExpectedTwoWay         string
		ExpectedTwoWayErr      string
		ExpectedTwoWayResult   string
		ExpectedThreeWay       string
		ExpectedThreeWayErr    string
		ExpectedThreeWayResult string
	}{
		// cases we can successfully strategically merge
		"no diff": {
			Original: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Current:  `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Modified: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,

			ExpectedTwoWay:         `{}`,
			ExpectedTwoWayResult:   `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			ExpectedThreeWay:       `{}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
		},
		"no diff even if modified null": {
			Original: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Current:  `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Modified: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{"key":null},"name":"foo","scalar":true}`,

			ExpectedTwoWay:         `{"complex_nullable":{"key":null}}`,
			ExpectedTwoWayResult:   `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{},"name":"foo","scalar":true}`,
			ExpectedThreeWay:       `{"complex_nullable":{"key":null}}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{},"name":"foo","scalar":true}`,
		},
		"discard nulls in nested and adds not nulls": {
			Original: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Current:  `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Modified: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{"key":{"keynotnull":"value","keynull":null}},"name":"foo","scalar":true}`,

			ExpectedTwoWay:         `{"complex_nullable":{"key":{"keynotnull":"value","keynull":null}}}`,
			ExpectedTwoWayResult:   `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{"key":{"keynotnull":"value"}},"name":"foo","scalar":true}`,
			ExpectedThreeWay:       `{"complex_nullable":{"key":{"keynotnull":"value","keynull":null}}}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":{"key":{"keynotnull":"value"}},"name":"foo","scalar":true}`,
		},
		"discard if modified all nulls": {
			Original: `{}`,
			Current:  `{}`,
			Modified: `{"complex":{"nested":null}}`,

			ExpectedTwoWay:         `{"complex":{"nested":null}}`,
			ExpectedTwoWayResult:   `{"complex":{}}`,
			ExpectedThreeWay:       `{"complex":{"nested":null}}`,
			ExpectedThreeWayResult: `{"complex":{}}`,
		},
		"add only not nulls": {
			Original: `{}`,
			Current:  `{}`,
			Modified: `{"complex":{"nested":null,"nested2":"foo"}}`,

			ExpectedTwoWay:         `{"complex":{"nested":null,"nested2":"foo"}}`,
			ExpectedTwoWayResult:   `{"complex":{"nested2":"foo"}}`,
			ExpectedThreeWay:       `{"complex":{"nested":null,"nested2":"foo"}}`,
			ExpectedThreeWayResult: `{"complex":{"nested2":"foo"}}`,
		},
		"null values in original are preserved": {
			Original: `{"thing":null}`,
			Current:  `{"thing":null}`,
			Modified: `{"nested":{"value":5},"thing":null}`,

			ExpectedTwoWay:         `{"nested":{"value":5}}`,
			ExpectedTwoWayResult:   `{"nested":{"value":5},"thing":null}`,
			ExpectedThreeWay:       `{"nested":{"value":5}}`,
			ExpectedThreeWayResult: `{"nested":{"value":5},"thing":null}`,
		},
		"nested null values in original are preserved": {
			Original: `{"complex":{"key":null},"thing":null}`,
			Current:  `{"complex":{"key":null},"thing":null}`,
			Modified: `{"complex":{"key":null},"nested":{"value":5},"thing":null}`,

			ExpectedTwoWay:         `{"nested":{"value":5}}`,
			ExpectedTwoWayResult:   `{"complex":{"key":null},"nested":{"value":5},"thing":null}`,
			ExpectedThreeWay:       `{"nested":{"value":5}}`,
			ExpectedThreeWayResult: `{"complex":{"key":null},"nested":{"value":5},"thing":null}`,
		},
		"add empty slices": {
			Original: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Current:  `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Modified: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":[],"name":"foo","scalar":true}`,

			ExpectedTwoWay:         `{"complex_nullable":[]}`,
			ExpectedTwoWayResult:   `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":[],"name":"foo","scalar":true}`,
			ExpectedThreeWay:       `{"complex_nullable":[]}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"complex":{"nested":true},"complex_nullable":[],"name":"foo","scalar":true}`,
		},
		"filter nulls from nested slices": {
			Original: `{}`,
			Current:  `{}`,
			Modified: `{"complex_nullable":[{"inner_one":{"key_one":"foo","key_two":null}}]}`,

			ExpectedTwoWay:         `{"complex_nullable":[{"inner_one":{"key_one":"foo","key_two":null}}]}`,
			ExpectedTwoWayResult:   `{"complex_nullable":[{"inner_one":{"key_one":"foo"}}]}`,
			ExpectedThreeWay:       `{"complex_nullable":[{"inner_one":{"key_one":"foo","key_two":null}}]}`,
			ExpectedThreeWayResult: `{"complex_nullable":[{"inner_one":{"key_one":"foo"}}]}`,
		},
		"filter if slice is all empty": {
			Original: `{}`,
			Current:  `{}`,
			Modified: `{"complex_nullable":[{"inner_one":{"key_one":null,"key_two":null}}]}`,

			ExpectedTwoWay:         `{"complex_nullable":[{"inner_one":{"key_one":null,"key_two":null}}]}`,
			ExpectedTwoWayResult:   `{"complex_nullable":[{"inner_one":{}}]}`,
			ExpectedThreeWay:       `{"complex_nullable":[{"inner_one":{"key_one":null,"key_two":null}}]}`,
			ExpectedThreeWayResult: `{"complex_nullable":[{"inner_one":{}}]}`,
		},
		"not filter nulls from non-associative slice": {
			Original: `{}`,
			Current:  `{}`,
			Modified: `{"complex_nullable":["key1",null,"key2"]}`,

			ExpectedTwoWay:         `{"complex_nullable":["key1",null,"key2"]}`,
			ExpectedTwoWayResult:   `{"complex_nullable":["key1",null,"key2"]}`,
			ExpectedThreeWay:       `{"complex_nullable":["key1",null,"key2"]}`,
			ExpectedThreeWayResult: `{"complex_nullable":["key1",null,"key2"]}`,
		},
		"added only": {
			Original: `{"name":"foo"}`,
			Current:  `{"name":"foo"}`,
			Modified: `{"name":"foo","scalar":true,"complex":{"nested":true},"array":[1,2,3]}`,

			ExpectedTwoWay:         `{"array":[1,2,3],"complex":{"nested":true},"scalar":true}`,
			ExpectedTwoWayResult:   `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			ExpectedThreeWay:       `{"array":[1,2,3],"complex":{"nested":true},"scalar":true}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
		},
		"removed only": {
			Original: `{"name":"foo","scalar":true,"complex":{"nested":true}}`,
			Current:  `{"name":"foo","scalar":true,"complex":{"nested":true},"array":[1,2,3]}`,
			Modified: `{"name":"foo"}`,

			ExpectedTwoWay:         `{"complex":null,"scalar":null}`,
			ExpectedTwoWayResult:   `{"name":"foo"}`,
			ExpectedThreeWay:       `{"complex":null,"scalar":null}`,
			ExpectedThreeWayResult: `{"array":[1,2,3],"name":"foo"}`,
		},

		// cases we cannot successfully strategically merge (expect errors)
		"diff": {
			Original: `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Current:  `{"array":[1,2,3],"complex":{"nested":true},"name":"foo","scalar":true}`,
			Modified: `{"array":[1,2,3],"complex":{"nested":false},"name":"foo","scalar":true}`,

			ExpectedTwoWayErr:   `unable to find api field`,
			ExpectedThreeWayErr: `unable to find api field`,
		},

		"json": {
			Original: `{"name":"foo","jsonItem":{"nested":{"nested2":{"nested3":{}}}}}`,
			Current:  `{"name":"foo","jsonItem":{"nested":{"nested2":{"nested3":{}}}}}`,
			Modified: `{"name":"foo","jsonItem":{"nested":{"nested2":{"nested3":{"a":"b"}}}}}`,

			ExpectedTwoWay:         `{"jsonItem":{"nested":{"nested2":{"nested3":{"a":"b"}}}}}`,
			ExpectedTwoWayResult:   `{"jsonItem":{"nested":{"nested2":{"nested3":{"a":"b"}}}},"name":"foo"}`,
			ExpectedThreeWay:       `{"jsonItem":{"nested":{"nested2":{"nested3":{"a":"b"}}}}}`,
			ExpectedThreeWayResult: `{"jsonItem":{"nested":{"nested2":{"nested3":{"a":"b"}}}},"name":"foo"}`,
			ExpectedTwoWayErr:      `unable to find api field`,
			ExpectedThreeWayErr:    `unable to find api field`,
		},
	}

	schemas := []LookupPatchMeta{
		mergeItemStructSchema,
	}

	keys := make([]string, 0, len(testcases))
	for k := range testcases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		t.Run(k, func(t *testing.T) {
			tc := testcases[k]
			for _, schema := range schemas {
				t.Run(schema.Name()+"/TwoWay", func(t *testing.T) {
					// Apply half: with the struct schema upstream stops at patch
					// creation whenever an error is expected, so nothing is applied.
					if len(tc.ExpectedTwoWayErr) > 0 {
						return
					}
					twoWay := []byte(tc.ExpectedTwoWay)

					twoWayResult, err := StrategicMergePatchUsingLookupPatchMeta([]byte(tc.Original), twoWay, schema)
					if err != nil {
						t.Errorf("using %s in testcase %s: error applying two-way patch: %v", getSchemaType(schema), k, err)
						return
					}
					if string(twoWayResult) != tc.ExpectedTwoWayResult {
						t.Errorf("using %s in testcase %s: expected two-way result:\n\t%s\ngot\n\t%s", getSchemaType(schema), k, string(tc.ExpectedTwoWayResult), string(twoWayResult))
						return
					}
				})

				t.Run(schema.Name()+"/ThreeWay", func(t *testing.T) {
					if len(tc.ExpectedThreeWayErr) > 0 {
						return
					}
					threeWay := []byte(tc.ExpectedThreeWay)

					threeWayResult, err := StrategicMergePatchUsingLookupPatchMeta([]byte(tc.Current), threeWay, schema)
					if err != nil {
						t.Errorf("using %s in testcase %s: error applying three-way patch: %v", getSchemaType(schema), k, err)
						return
					} else if string(threeWayResult) != tc.ExpectedThreeWayResult {
						t.Errorf("using %s in testcase %s: expected three-way result:\n\t%s\ngot\n\t%s", getSchemaType(schema), k, string(tc.ExpectedThreeWayResult), string(threeWayResult))
						return
					}
				})
			}
		})
	}
}
