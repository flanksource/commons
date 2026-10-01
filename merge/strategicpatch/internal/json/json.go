/*
Copyright 2015 The Kubernetes Authors.

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
// Ported from k8s.io/apimachinery v0.36.2 pkg/util/json/json.go; Unmarshal reimplemented on encoding/json.

package json

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Marshal delegates to json.Marshal
// It is only here so this package can be a drop-in for common encoding/json uses
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// limit recursive depth to prevent stack overflow errors
const maxDepth = 10000

// Unmarshal unmarshals the given data.
// Object keys are case-sensitive.
// Numbers decoded into interface{} fields are converted to int64 or float64.
//
// Upstream delegates to sigs.k8s.io/json's case-sensitive, int-preserving
// decoder. Here encoding/json decodes with UseNumber and the conversion below
// turns each json.Number into int64 or float64 the same way. encoding/json
// matches struct fields case-insensitively, so only the untyped targets the
// patch code decodes into are accepted; any other target is an error rather
// than a silently different decode.
func Unmarshal(data []byte, v interface{}) error {
	if !json.Valid(data) {
		// Decode again only to surface encoding/json's syntax error.
		return json.Unmarshal(data, new(interface{}))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	switch target := v.(type) {
	case *map[string]interface{}:
		if err := decoder.Decode(target); err != nil {
			return err
		}
		return ConvertMapNumbers(*target, 0)
	case *[]interface{}:
		if err := decoder.Decode(target); err != nil {
			return err
		}
		return ConvertSliceNumbers(*target, 0)
	case *interface{}:
		if err := decoder.Decode(target); err != nil {
			return err
		}
		return ConvertInterfaceNumbers(target, 0)
	default:
		return fmt.Errorf("json: cannot unmarshal into %T: only *map[string]interface{}, *[]interface{} and *interface{} decode case-sensitively", v)
	}
}

// ConvertInterfaceNumbers converts any json.Number values to int64 or float64.
// Values which are map[string]interface{} or []interface{} are recursively visited
func ConvertInterfaceNumbers(v *interface{}, depth int) error {
	var err error
	switch v2 := (*v).(type) {
	case json.Number:
		*v, err = convertNumber(v2)
	case map[string]interface{}:
		err = ConvertMapNumbers(v2, depth+1)
	case []interface{}:
		err = ConvertSliceNumbers(v2, depth+1)
	}
	return err
}

// ConvertMapNumbers traverses the map, converting any json.Number values to int64 or float64.
// values which are map[string]interface{} or []interface{} are recursively visited
func ConvertMapNumbers(m map[string]interface{}, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("exceeded max depth of %d", maxDepth)
	}

	var err error
	for k, v := range m {
		switch v := v.(type) {
		case json.Number:
			m[k], err = convertNumber(v)
		case map[string]interface{}:
			err = ConvertMapNumbers(v, depth+1)
		case []interface{}:
			err = ConvertSliceNumbers(v, depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ConvertSliceNumbers traverses the slice, converting any json.Number values to int64 or float64.
// values which are map[string]interface{} or []interface{} are recursively visited
func ConvertSliceNumbers(s []interface{}, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("exceeded max depth of %d", maxDepth)
	}

	var err error
	for i, v := range s {
		switch v := v.(type) {
		case json.Number:
			s[i], err = convertNumber(v)
		case map[string]interface{}:
			err = ConvertMapNumbers(v, depth+1)
		case []interface{}:
			err = ConvertSliceNumbers(v, depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// convertNumber converts a json.Number to an int64 or float64, or returns an error
func convertNumber(n json.Number) (interface{}, error) {
	// Attempt to convert to an int64 first
	if i, err := n.Int64(); err == nil {
		return i, nil
	}
	// Return a float64 (default json.Decode() behavior)
	// An overflow will return an error
	return n.Float64()
}
