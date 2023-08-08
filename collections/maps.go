package collections

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flanksource/commons/files"
)

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

func MapKeys[K comparable](m map[K]any) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}
