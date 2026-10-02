package merge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/flanksource/commons/merge/strategicpatch"
)

// The struct tags Kubernetes API types use to declare how a list is patched.
const (
	patchStrategyTag = "patchStrategy"
	patchMergeKeyTag = "patchMergeKey"
)

// The patchStrategy values Kubernetes' strategic merge patch understands.
const (
	strategyMerge      = "merge"
	strategyReplace    = "replace"
	strategyRetainKeys = "retainKeys"
)

// patchStrategic reports whether a field declares a Kubernetes patch strategy,
// and rejects the declarations that cannot mean what they say.
//
// A struct tagged only retainKeys (e.g. appsv1.DeploymentSpec.Strategy) is not
// strategic here: without a $retainKeys directive in the patch, a strategic
// merge patch merges it field by field, which is what the structural walk does.
func patchStrategic(field reflect.StructField, path string) bool {
	strategy, hasStrategy := field.Tag.Lookup(patchStrategyTag)
	_, hasKey := field.Tag.Lookup(patchMergeKeyTag)
	if !hasStrategy && !hasKey {
		return false
	}
	if !hasStrategy {
		panic(fmt.Sprintf("merge: %s declares %s but no %s", path, patchMergeKeyTag, patchStrategyTag))
	}
	for _, s := range strings.Split(strategy, ",") {
		if s != strategyMerge && s != strategyReplace && s != strategyRetainKeys {
			panic(fmt.Sprintf("merge: %s is tagged %s:%q, but %q is not one of %s, %s or %s",
				path, patchStrategyTag, strategy, s, strategyMerge, strategyReplace, strategyRetainKeys))
		}
	}
	if strategy == strategyRetainKeys && !hasKey && isStruct(field.Type) {
		return false
	}
	switch {
	case field.Type.Kind() != reflect.Slice:
		panic(fmt.Sprintf("merge: %s is tagged %s:%q but is a %s, not a list",
			path, patchStrategyTag, strategy, field.Type.Kind()))
	}
	if rule, tagged := field.Tag.Lookup(tagName); tagged {
		panic(fmt.Sprintf("merge: %s declares both %s:%q and %s:%q — a list merges by one rule",
			path, tagName, rule, patchStrategyTag, strategy))
	}
	return true
}

func isStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// strategicMerge applies the override's list to the base's as a Kubernetes
// strategic merge patch of the owning struct, and returns the result — or the
// zero Value when the override has no elements and the base should stand.
//
// Both lists are wrapped in a one-field JSON document keyed by the field's JSON
// name, so the owning struct type is the patch schema: the ported
// StrategicMergePatch reads this field's patchStrategy and patchMergeKey, and
// the element type's own tags for anything nested, exactly as it would on a
// Kubernetes object.
func strategicMerge(owner reflect.Type, field reflect.StructField, base, override reflect.Value, path string) reflect.Value {
	if override.Len() == 0 {
		return reflect.Value{}
	}
	key := jsonName(field, path)
	document := func(list reflect.Value) []byte {
		data, err := json.Marshal(map[string]any{key: list.Interface()})
		if err != nil {
			panic(fmt.Sprintf("merge: %s: encoding %s for strategic merge: %v", path, list.Type(), err))
		}
		return data
	}
	patched, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(
		document(base), document(override), strategicpatch.PatchMetaFromStruct{T: owner})
	if err != nil {
		panic(fmt.Sprintf("merge: %s: strategic merge: %v", path, err))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(patched, &fields); err != nil {
		panic(fmt.Sprintf("merge: %s: decoding strategic merge result %s: %v", path, patched, err))
	}
	merged := reflect.New(field.Type)
	if err := json.Unmarshal(fields[key], merged.Interface()); err != nil {
		panic(fmt.Sprintf("merge: %s: decoding strategic merge result %s into %s: %v", path, patched, field.Type, err))
	}
	return merged.Elem()
}

// jsonName is the key encoding/json writes a field under.
func jsonName(field reflect.StructField, path string) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	switch name {
	case "-":
		panic(fmt.Sprintf("merge: %s is tagged %s but json:\"-\" leaves it out of the patch", path, patchStrategyTag))
	case "":
		return field.Name
	default:
		return name
	}
}
