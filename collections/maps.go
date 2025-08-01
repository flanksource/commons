package collections

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ToGenericMap converts a map[string]string to a map[string]interface{}.
// Useful when you need to pass string maps to functions expecting generic maps.
//
// Example:
//
//	strMap := map[string]string{"name": "John", "age": "30"}
//	generic := collections.ToGenericMap(strMap)
func ToGenericMap(m map[string]string) map[string]interface{} {
	var out = map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ToStringMap converts a map[string]interface{} to a map[string]string.
// Each value is converted using fmt.Sprintf("%v", value).
//
// Example:
//
//	data := map[string]interface{}{"count": 42, "active": true}
//	strings := collections.ToStringMap(data)
//	// Result: {"count": "42", "active": "true"}
func ToStringMap(m map[string]interface{}) map[string]string {
	var out = make(map[string]string)
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// ToBase64Map converts a map[string]interface{} to a map[string]string.
// []byte values are encoded as base64, other types use fmt.Sprintf.
//
// Example:
//
//	data := map[string]interface{}{
//		"text": "hello",
//		"binary": []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f},
//	}
//	encoded := collections.ToBase64Map(data)
//	// Result: {"text": "hello", "binary": "SGVsbG8="}
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

// ToByteMap converts a map[string]interface{} to a map[string][]byte.
// String values are converted to []byte, []byte values are kept as-is.
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

// MergeMap merges two maps, with values from b taking precedence.
// Returns the merged map (modifies map a in place).
//
// Example:
//
//	defaults := map[string]int{"timeout": 30, "retries": 3}
//	overrides := map[string]int{"timeout": 60}
//	config := collections.MergeMap(defaults, overrides)
//	// Result: {"timeout": 60, "retries": 3}
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

// KeyValueSliceToMap converts a slice of "key=value" strings to a map.
// Strings without '=' are treated as keys with empty values.
//
// Example:
//
//	args := []string{"env=prod", "debug=true", "verbose"}
//	config := collections.KeyValueSliceToMap(args)
//	// Result: {"env": "prod", "debug": "true", "verbose": ""}
func KeyValueSliceToMap(in []string) map[string]string {
	sanitized := make(map[string]string, len(in))
	for _, item := range in {
		splits := strings.SplitN(item, "=", 2)
		if len(splits) == 1 {
			// For no keys, we add an empty string to match just the key
			splits = append(splits, "")
		}
		sanitized[strings.TrimSpace(splits[0])] = strings.TrimSpace(splits[1])
	}
	return sanitized
}

// SelectorToMap parses a comma-separated selector string into a map.
// Commonly used for Kubernetes-style label selectors.
//
// Example:
//
//	selector := "app=nginx,env=prod,tier=frontend"
//	labels := collections.SelectorToMap(selector)
//	// Result: {"app": "nginx", "env": "prod", "tier": "frontend"}
func SelectorToMap(selector string) (matchLabels map[string]string) {
	labels := strings.Split(selector, ",")
	return KeyValueSliceToMap(labels)
}

// SortedMap converts a map to a sorted key=value string representation.
// Keys are sorted alphabetically for consistent output.
//
// Example:
//
//	labels := map[string]string{"tier": "web", "app": "nginx"}
//	result := collections.SortedMap(labels)
//	// Result: "app=nginx,tier=web"
func SortedMap(m map[string]string) string {
	keys := MapKeys(m)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var items []string
	for _, k := range keys {
		items = append(items, fmt.Sprintf("%s=%s", k, m[k]))
	}

	return strings.Join(items, ",")
}

// MapToIni converts a map to INI format with one key=value pair per line.
//
// Example:
//
//	config := map[string]string{"host": "localhost", "port": "8080"}
//	ini := collections.MapToIni(config)
//	// Result: "host=localhost\nport=8080\n"
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

	ini, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(ini), "\n") {
		values := strings.Split(line, "=")
		if len(values) == 2 {
			result[values[0]] = values[1]
		}
	}
	return result
}

// MapKeys returns a slice containing all keys from the map.
// The order of keys is non-deterministic.
//
// Example:
//
//	users := map[int]string{1: "Alice", 2: "Bob", 3: "Charlie"}
//	ids := collections.MapKeys(users) // [1, 2, 3] (order may vary)
func MapKeys[K comparable, T any](m map[K]T) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}
