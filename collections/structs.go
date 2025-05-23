package collections

import (
	"encoding/json"
	"reflect"
)

// StructToMap takes an object and returns all it's field in a map
func StructToMap(s interface{}) map[string]interface{} {
	values := make(map[string]interface{})
	value := reflect.ValueOf(s)

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.CanInterface() {
			v := field.Interface()
			if v != nil && v != "" {
				values[value.Type().Field(i).Name] = v
			}
		}
	}
	return values
}

// ToJSONMap takes an input value of struct or map type and converts it to a map[string]any representation
// using JSON encoding and decoding.
func ToJSONMap(s any) (map[string]any, error) {
	var raw []byte
	var err error

	switch s := s.(type) {
	case string:
		raw = []byte(s)
	case []byte:
		raw = s
	default:
		raw, err = json.Marshal(s)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[string]any)
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// StructToJSON takes an object and returns its json form
func StructToJSON(v any) (string, error) {
	b, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// StructToIni takes an object and serializes it's fields in INI format
func StructToIni(s interface{}) string {
	str := ""
	for k, v := range StructToMap(s) {
		str += k + "=" + ToString(v) + "\n"
	}
	return str
}

// MergeStructs merges two structs where patch is applied on top of base
func MergeStructs[T any](base, patch T) (T, error) {
	jb, err := json.Marshal(patch)
	if err != nil {
		return base, err
	}
	err = json.Unmarshal(jb, &base)
	if err != nil {
		return base, err
	}

	return base, nil
}
