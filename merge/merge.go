// Package merge layers one configuration value over another, and deep-copies
// one, from the struct definitions themselves.
//
// It exists to delete a category of code: hand-written `if o.X != "" { s.X = o.X }`
// chains, hand-written emptiness tests, and hand-written field-by-field copies.
// Each of those has to be updated whenever a field is added, and each is a place
// a field can be silently forgotten. Structural merging cannot forget a field,
// so adding one to a spec needs no merge code at all.
//
// The default behaviour is:
//
//   - scalars and strings — the override wins when it is set (a zero value means
//     "unset", so an override can turn a bool on but never off);
//   - slices — replaced wholesale when the override's is non-empty, because a
//     list of tools means exactly those tools;
//   - maps — merged key-wise, because a map is a set of independent settings;
//   - structs, including those behind a pointer — merged field by field, so
//     setting one sub-field does not erase its siblings.
//
// A single field can name an exception the structure cannot express, with a tag:
//
//	Pre   []string `merge:"append"`        // the base's elements, then the override's
//	Allow []string `merge:"append,unique"` // as append, with repeats dropped
//
// Tags are read on struct fields reached through structs and through pointers
// that are non-nil on both sides. A tag that cannot mean what it says — append on
// something that is not a list, unique over a non-comparable element type, a rule
// this package does not define — panics rather than being ignored, because a
// silently ignored tag reads exactly like an honoured one.
//
// Policy names the exceptions that belong to a whole type rather than to one
// field, including the types whose merge is a domain rule rather than a
// structural one and which therefore merge themselves.
package merge

import (
	"fmt"
	"maps"
	"reflect"

	"dario.cat/mergo"
)

// tagName is the struct tag namespace for per-field merge rules.
const tagName = "merge"

// Policy declares the exceptions to structural merging for one family of types.
// The zero Policy is valid and means "no exceptions".
type Policy struct {
	// Replace lists types that are indivisible: when the override's value is set
	// it replaces the base's wholesale instead of being merged into. A pointer
	// type counts as set when it is non-nil — which is how an explicit
	// *float64(0) or *bool(false) survives a merge that would otherwise read the
	// pointed-at zero as "unset" and drop it.
	//
	// Name a type by its zero value: merge.Policy{Replace: []any{(*float64)(nil)}}.
	Replace []any

	// Shared lists types that are referenced, not owned: Apply assigns them like
	// Replace, and Clone copies the reference instead of walking it. Registries,
	// catalogs and caches belong here — deep-copying one is both wasteful and
	// wrong, because consumers rely on pointer identity.
	Shared []any

	// Merger lists types that merge themselves. Each must declare the method
	// `Merge(T) T` on the type named; Apply calls base.Merge(override) and takes
	// the result instead of walking the type. It is where a merge rule that is a
	// domain decision rather than a structural one lives: a list that accumulates
	// across configuration layers instead of being replaced, or two fields that
	// only mean anything when they move together.
	//
	// The method always runs — a nil slice or map destination is materialised
	// first, so "the base had none" reaches Merge as an empty value rather than
	// silently bypassing it. A pointer type therefore cannot be a Merger (a nil
	// pointer has nothing to call the method on), and naming a type without that
	// exact method panics rather than quietly reverting to structural merging.
	Merger []any
}

// With returns the union of two policies, so a nested type's policy can be
// composed into its container's rather than restated.
func (p Policy) With(other Policy) Policy {
	return Policy{
		Replace: append(append([]any{}, p.Replace...), other.Replace...),
		Shared:  append(append([]any{}, p.Shared...), other.Shared...),
		Merger:  append(append([]any{}, p.Merger...), other.Merger...),
	}
}

// Apply returns base with override's set fields layered on top, per the package
// default and policy's exceptions. Neither argument is mutated, and the result
// shares no mutable memory with either — a merged spec can be edited without
// reaching back into the config it inherited from.
//
// T must be a struct, map or slice type; anything else is a caller error and
// panics, as does a mergo failure, which cannot occur for a well-formed T since
// both operands are the same concrete type by construction.
func Apply[T any](base, override T, policy Policy) T {
	mergers := mergerSet(policy.Merger)
	assign := typeSet(append(append([]any{}, policy.Replace...), policy.Shared...))
	out := clone(base, policy, mergers)
	src := clone(override, policy, mergers)

	accumulate(reflect.ValueOf(&out).Elem(), reflect.ValueOf(&src).Elem(),
		union(assign, mergers), fmt.Sprintf("%T", base))

	opts := []func(*mergo.Config){mergo.WithOverride}
	if len(assign) > 0 || len(mergers) > 0 {
		opts = append(opts, mergo.WithTransformers(policyTransformer{assign: assign, mergers: mergers}))
	}
	if err := mergo.Merge(&out, src, opts...); err != nil {
		panic(fmt.Sprintf("merge %T: %v", base, err))
	}
	return out
}

// accumulate rewrites each of the override's `append`-tagged slices to the
// concatenation of the base's and its own, ahead of the merge. The default rule —
// a non-empty override slice replaces the base's — then lands the accumulated
// value, so the tag needs no machinery of its own and cannot interact with the
// policy types, which own their rule and are not walked.
func accumulate(base, override reflect.Value, owned map[reflect.Type]bool, path string) {
	if owned[base.Type()] {
		return
	}
	switch base.Kind() {
	case reflect.Pointer:
		if !base.IsNil() && !override.IsNil() {
			accumulate(base.Elem(), override.Elem(), owned, path)
		}
	case reflect.Struct:
		for i := range base.NumField() {
			field := base.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			fieldPath := path + "." + field.Name
			tag, tagged := field.Tag.Lookup(tagName)
			if !tagged {
				accumulate(base.Field(i), override.Field(i), owned, fieldPath)
				continue
			}
			if base.Field(i).Kind() != reflect.Slice {
				panic(fmt.Sprintf("merge: %s is tagged %s:%q but is a %s, not a list",
					fieldPath, tagName, tag, base.Field(i).Kind()))
			}
			if merged := concat(base.Field(i), override.Field(i), unique(tag, fieldPath), fieldPath); merged.IsValid() {
				override.Field(i).Set(merged)
			}
		}
	}
}

// unique reports whether the tag asks for repeats to be dropped, and rejects any
// rule this package does not define.
func unique(tag, path string) bool {
	switch tag {
	case "append":
		return false
	case "append,unique":
		return true
	default:
		panic(fmt.Sprintf("merge: %s declares %s:%q, want %q or %q", path, tagName, tag, "append", "append,unique"))
	}
}

// concat returns the base's elements followed by the override's, or the zero
// Value when neither layer contributed one and the field should be left as it is.
func concat(base, override reflect.Value, unique bool, path string) reflect.Value {
	if base.Len() == 0 && override.Len() == 0 {
		return reflect.Value{}
	}
	if unique && !base.Type().Elem().Comparable() {
		panic(fmt.Sprintf("merge: %s asks for %s:%q but its element type %s is not comparable",
			path, tagName, "append,unique", base.Type().Elem()))
	}
	out := reflect.MakeSlice(base.Type(), 0, base.Len()+override.Len())
	seen := map[any]bool{}
	for _, layer := range []reflect.Value{base, override} {
		for i := range layer.Len() {
			if unique {
				key := layer.Index(i).Interface()
				if seen[key] {
					continue
				}
				seen[key] = true
			}
			out = reflect.Append(out, layer.Index(i))
		}
	}
	return out
}

// union is the set of types that declare their own merge rule, and are therefore
// neither walked for field tags nor merged structurally.
func union(sets ...map[reflect.Type]bool) map[reflect.Type]bool {
	out := map[reflect.Type]bool{}
	for _, set := range sets {
		maps.Copy(out, set)
	}
	return out
}

// Clone returns a deep copy of v. Interfaces, functions and channels are copied
// by reference — they carry behaviour rather than configuration — as are the
// types named in policy.Shared. Unexported fields are copied as they stand.
func Clone[T any](v T, policy Policy) T {
	return clone(v, policy, nil)
}

// clone is Clone plus Apply's extra duty: materialise the nil slices and maps
// whose type declares its own Merge, so mergo — which skips its transformers for
// a nil destination — cannot bypass one.
func clone[T any](v T, policy Policy, materialize map[reflect.Type]bool) T {
	shared := typeSet(policy.Shared)
	if len(shared) == 0 {
		shared = nil
	}
	return cloneValue(reflect.ValueOf(&v).Elem(), shared, materialize).Interface().(T)
}

func cloneValue(v reflect.Value, shared, materialize map[reflect.Type]bool) reflect.Value {
	if !v.IsValid() || shared[v.Type()] {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(cloneValue(v.Elem(), shared, materialize))
		return out
	case reflect.Slice:
		if v.IsNil() {
			if !materialize[v.Type()] {
				return v
			}
			return reflect.MakeSlice(v.Type(), 0, 0)
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(cloneValue(v.Index(i), shared, materialize))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			if !materialize[v.Type()] {
				return v
			}
			return reflect.MakeMapWithSize(v.Type(), 0)
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		for iter := v.MapRange(); iter.Next(); {
			out.SetMapIndex(cloneValue(iter.Key(), shared, materialize), cloneValue(iter.Value(), shared, materialize))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		for i := range v.Len() {
			out.Index(i).Set(cloneValue(v.Index(i), shared, materialize))
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				out.Field(i).Set(cloneValue(v.Field(i), shared, materialize))
			}
		}
		return out
	default:
		return v
	}
}

// policyTransformer is the mergo hook that short-circuits every policy type,
// keyed on the destination's type.
type policyTransformer struct {
	assign  map[reflect.Type]bool
	mergers map[reflect.Type]bool
}

func (t policyTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	switch {
	case t.mergers[typ]:
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(dst.MethodByName("Merge").Call([]reflect.Value{src})[0])
			}
			return nil
		}
	case t.assign[typ]:
		return func(dst, src reflect.Value) error {
			if isSet(src) && dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	default:
		return nil
	}
}

// isSet reports whether an override value carries an instruction. A pointer is
// set when it is non-nil, so a pointer to a zero scalar still wins; every other
// kind is set when it is not the zero value.
func isSet(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return !v.IsNil()
	default:
		return !v.IsZero()
	}
}

// mergerSet validates that every Merger type can actually merge itself. The
// signature is checked here rather than at the merge, because a type whose method
// was renamed or re-typed would otherwise fall back to structural merging — the
// silent drift this package exists to remove.
func mergerSet(values []any) map[reflect.Type]bool {
	set := typeSet(values)
	for typ := range set {
		if typ.Kind() == reflect.Pointer {
			panic(fmt.Sprintf("merge: policy Merger type %s must not be a pointer — a nil pointer has no value to merge into", typ))
		}
		method, ok := typ.MethodByName("Merge")
		if !ok || method.Type.NumIn() != 2 || method.Type.In(1) != typ ||
			method.Type.NumOut() != 1 || method.Type.Out(0) != typ {
			panic(fmt.Sprintf("merge: policy Merger type %s must declare a method Merge(%s) %s", typ, typ, typ))
		}
	}
	return set
}

func typeSet(values []any) map[reflect.Type]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[reflect.Type]bool, len(values))
	for _, value := range values {
		typ := reflect.TypeOf(value)
		if typ == nil {
			panic("merge: policy type must be named by a typed zero value, e.g. (*float64)(nil)")
		}
		set[typ] = true
	}
	return set
}
