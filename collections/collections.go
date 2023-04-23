package collections

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/flanksource/commons/files"
)

func takeSliceArg(arg interface{}) (out []interface{}, ok bool) {
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Slice {
		return nil, false
	}

	c := val.Len()
	out = make([]interface{}, c)
	for i := 0; i < val.Len(); i++ {
		out[i] = val.Index(i).Interface()
	}
	return out, true
}

// ToString takes an object and tries to convert it to a string
func ToString(i interface{}) string {
	if slice, ok := takeSliceArg(i); ok {
		s := ""
		for _, v := range slice {
			if s != "" {
				s += ", "
			}
			s += ToString(v)
		}
		return s

	}
	switch v := i.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case interface{}:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	default:
		// panic(fmt.Sprintf("I don't know about type %T!\n", v))
	}
	return ""
}

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

// MapToIni takes a map and converts it into an INI formatted string
func MapToIni(Map map[string]string) string {
	str := ""
	for k, v := range Map {
		str += k + "=" + ToString(v) + "\n"
	}
	return str
}

// IniToMap takes the path to an INI formatted file and transforms it into a map
func IniToMap(path string) map[string]string {
	result := make(map[string]string)
	ini := files.SafeRead(path)
	for _, line := range strings.Split(ini, "\n") {
		values := strings.Split(line, "=")
		if len(values) == 2 {
			result[values[0]] = values[1]
		}
	}
	return result
}

// ReplaceAllInSlice runs strings.Replace on all elements in a slice and returns the result
func ReplaceAllInSlice(a []string, find string, replacement string) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.Replace(s, find, replacement, -1))
	}
	return
}

// SplitAllInSlice runs strings.Split on all elements in a slice and returns the results at the given index
func SplitAllInSlice(a []string, split string, index int) (replaced []string) {
	for _, s := range a {
		replaced = append(replaced, strings.Split(s, split)[index])
	}
	return
}

// ToGenericMap converts a map[string]string to a map[string]interface{}
func ToGenericMap(m map[string]string) map[string]interface{} {
	var out = map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ToStringMap converts a map[string]interface{} to a map[string]string by using each objects String() fn
func ToStringMap(m map[string]interface{}) map[string]string {
	var out = make(map[string]string)
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// ToBase64Map converts a map[string]interface{} to a map[string]string by converting []byte to base64
func ToBase64Map(m map[string]interface{}) map[string]string {
	var out = make(map[string]string)
	for k, v := range m {
		switch b := v.(type) {
		case []byte:
			out[k] = base64.StdEncoding.EncodeToString(b)
		default:
			out[k] = fmt.Sprintf("%v", v)
		}
	}
	return out
}

// ToByteMap converts a map[string]interface{} to a map[string]string by converting []byte to base64
func ToByteMap(m map[string]interface{}) map[string][]byte {
	var out = make(map[string][]byte)
	for k, v := range m {
		switch b := v.(type) {
		case []byte:
			out[k] = b
		default:
			out[k] = []byte(fmt.Sprintf("%v", v))
		}
	}
	return out
}

// Find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// Contains tells whether a contains x.
func Contains[T comparable](a []T, x T) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// MergeMap will merge map b into a.
// On key collision, map b takes precedence.
func MergeMap[T1 comparable, T2 any](a, b map[T1]T2) map[T1]T2 {
	if a == nil {
		a = make(map[T1]T2)
	}

	if b == nil {
		b = make(map[T1]T2)
	}

	for k, v := range b {
		a[k] = v
	}

	return a
}

// KeyValueSliceToMap takes in a list of strings in a=b format
// and returns a map from it.
//
// Any string that's not in a=b format will be ignored.
func KeyValueSliceToMap(in []string) map[string]string {
	sanitized := make(map[string]string, len(in))
	for _, item := range in {
		splits := strings.SplitN(item, "=", 2)
		if len(splits) == 1 {
			continue // ignore this item. not in a=b format
		}

		sanitized[strings.TrimSpace(splits[0])] = strings.TrimSpace(splits[1])
	}

	return sanitized
}

// matchItems returns true if any of the items in the list match the item
// negative matches are supported by prefixing the item with a !
// * matches everything
func MatchItems(item string, items ...string) bool {
	if len(items) == 0 {
		return true
	}

	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			if item == strings.TrimPrefix(i, "!") {
				return false
			}
		}
	}

	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			continue
		}
		if i == "*" || item == i {
			return true
		}
	}
	return false
}
