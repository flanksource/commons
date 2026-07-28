package merge_test

import (
	"reflect"
	"testing"

	"github.com/flanksource/commons/merge"
)

// catalog stands in for a registry: a large shared structure reached by identity,
// which must be referenced rather than walked.
type catalog struct{ Entries []string }

type limits struct {
	Cost    float64
	Retries *int
}

type config struct {
	Name     string
	Verbose  bool
	Tags     []string
	Settings map[string]string
	Limits   limits
	Nested   *limits
	Rate     *float64
	Catalog  *catalog
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func TestApply_Defaults(t *testing.T) {
	base := config{
		Name:     "base",
		Tags:     []string{"a", "b"},
		Settings: map[string]string{"one": "1", "two": "2"},
		Limits:   limits{Cost: 5, Retries: intPtr(3)},
	}
	override := config{
		Verbose:  true,
		Tags:     []string{"c"},
		Settings: map[string]string{"two": "II"},
		Limits:   limits{Cost: 9},
	}

	got := merge.Apply(base, override, merge.Policy{})

	if got.Name != "base" {
		t.Errorf("Name = %q, want the base's kept when the override is silent", got.Name)
	}
	if !got.Verbose {
		t.Error("Verbose = false, want an override able to turn a flag on")
	}
	if !reflect.DeepEqual(got.Tags, []string{"c"}) {
		t.Errorf("Tags = %v, want the override's list wholesale", got.Tags)
	}
	if !reflect.DeepEqual(got.Settings, map[string]string{"one": "1", "two": "II"}) {
		t.Errorf("Settings = %v, want a key-wise merge", got.Settings)
	}
	// The override speaks about Cost only, so Retries — its sibling — survives.
	if got.Limits.Cost != 9 || got.Limits.Retries == nil || *got.Limits.Retries != 3 {
		t.Errorf("Limits = %+v, want Cost overridden and Retries kept", got.Limits)
	}
}

// A false or 0 in the override is indistinguishable from absent, so it must not
// erase the base. This is what makes a layered config predictable, and it is why
// an explicitly-zero value needs a pointer plus a Replace policy (below).
func TestApply_ZeroMeansUnset(t *testing.T) {
	base := config{Name: "base", Verbose: true, Limits: limits{Cost: 5}}

	got := merge.Apply(base, config{}, merge.Policy{})

	if got.Name != "base" || !got.Verbose || got.Limits.Cost != 5 {
		t.Errorf("merged = %+v, want the base unchanged by an empty override", got)
	}
}

// The case a scalar pointer exists to express: an explicit zero. Without the
// Replace policy the engine would merge through the pointer, read the pointed-at
// 0 as unset, and silently keep the base — reintroducing the bug the pointer was
// added to prevent.
func TestApply_ReplaceKeepsExplicitZero(t *testing.T) {
	policy := merge.Policy{Replace: []any{(*float64)(nil)}}
	base := config{Rate: floatPtr(0.7)}

	if got := merge.Apply(base, config{Rate: floatPtr(0)}, policy); got.Rate == nil || *got.Rate != 0 {
		t.Errorf("Rate = %v, want an explicit 0 to win", got.Rate)
	}
	if got := merge.Apply(base, config{}, policy); got.Rate == nil || *got.Rate != 0.7 {
		t.Errorf("Rate = %v, want a nil override to leave the base alone", got.Rate)
	}
	if got := merge.Apply(config{}, config{Rate: floatPtr(0)}, policy); got.Rate == nil || *got.Rate != 0 {
		t.Errorf("Rate = %v, want an explicit 0 to reach a base that had none", got.Rate)
	}
}

// A shared type is referenced, not owned: consumers rely on pointer identity, and
// walking one would copy an entire catalog per merge.
func TestApply_SharedIsReferencedNotCopied(t *testing.T) {
	policy := merge.Policy{Shared: []any{(*catalog)(nil)}}
	shared := &catalog{Entries: []string{"claude", "codex"}}

	got := merge.Apply(config{Catalog: shared}, config{}, policy)
	if got.Catalog != shared {
		t.Error("Catalog pointer changed, want the same instance shared through the merge")
	}

	replacement := &catalog{Entries: []string{"gemini"}}
	if got := merge.Apply(config{Catalog: shared}, config{Catalog: replacement}, policy); got.Catalog != replacement {
		t.Error("Catalog = base's, want the override's instance")
	}
}

// Purity is the property that lets a merged value be edited freely: it must not
// reach back into the config it inherited from, on either side.
func TestApply_ResultSharesNoMemory(t *testing.T) {
	base := config{Tags: []string{"a"}, Settings: map[string]string{"one": "1"}}
	override := config{Nested: &limits{Cost: 5, Retries: intPtr(3)}}

	got := merge.Apply(base, override, merge.Policy{})
	got.Tags[0] = "mutated"
	got.Settings["one"] = "mutated"
	got.Nested.Cost = 99
	*got.Nested.Retries = 99

	if base.Tags[0] != "a" || base.Settings["one"] != "1" {
		t.Errorf("base mutated through the result: %+v", base)
	}
	if override.Nested.Cost != 5 || *override.Nested.Retries != 3 {
		t.Errorf("override mutated through the result: %+v", override.Nested)
	}
}

// A nil destination pointer is the case mergo would otherwise satisfy by
// assigning the source pointer itself; Apply clones first so the result still
// owns its memory.
func TestApply_NilDestinationPointerIsNotAliased(t *testing.T) {
	override := config{Nested: &limits{Cost: 5}}

	got := merge.Apply(config{}, override, merge.Policy{})
	if got.Nested == override.Nested {
		t.Fatal("result aliases the override's pointer")
	}
	got.Nested.Cost = 99
	if override.Nested.Cost != 5 {
		t.Errorf("override mutated through the result: %+v", override.Nested)
	}
}

func TestClone_IsDeepExceptShared(t *testing.T) {
	shared := &catalog{Entries: []string{"claude"}}
	original := config{
		Tags:     []string{"a"},
		Settings: map[string]string{"one": "1"},
		Nested:   &limits{Cost: 5, Retries: intPtr(3)},
		Catalog:  shared,
	}

	got := merge.Clone(original, merge.Policy{Shared: []any{(*catalog)(nil)}})
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("clone = %+v, want an equal value", got)
	}
	if got.Catalog != shared {
		t.Error("Catalog was copied, want the shared instance referenced")
	}

	got.Tags[0] = "mutated"
	got.Settings["one"] = "mutated"
	*got.Nested.Retries = 99
	if original.Tags[0] != "a" || original.Settings["one"] != "1" || *original.Nested.Retries != 3 {
		t.Errorf("original mutated through the clone: %+v", original)
	}
}

// tagList accumulates across layers and drops repeats. That is a statement about
// what a list of tags means, which no structural walk could infer — the whole
// reason a type gets to merge itself.
type tagList []string

func (base tagList) Merge(override tagList) tagList {
	seen := make(map[string]bool, len(base)+len(override))
	var out tagList
	for _, tag := range append(append(tagList{}, base...), override...) {
		if seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

type layered struct {
	Name string
	Tags tagList
}

func TestApply_MergerOwnsItsRule(t *testing.T) {
	policy := merge.Policy{Merger: []any{tagList(nil)}}
	base := layered{Name: "base", Tags: tagList{"a", "b"}}

	got := merge.Apply(base, layered{Tags: tagList{"b", "c"}}, policy)

	if !reflect.DeepEqual(got.Tags, tagList{"a", "b", "c"}) {
		t.Errorf("Tags = %v, want the layers accumulated and deduped, not replaced", got.Tags)
	}
	if !reflect.DeepEqual(base.Tags, tagList{"a", "b"}) {
		t.Errorf("base mutated through the merge: %v", base.Tags)
	}
}

// mergo skips its transformers for a nil destination, which would let "the base
// had no tags" quietly bypass the rule and assign the override wholesale — repeats
// and all. Apply materialises the nil first, so Merge always gets to speak.
func TestApply_MergerRunsAgainstANilBase(t *testing.T) {
	policy := merge.Policy{Merger: []any{tagList(nil)}}

	got := merge.Apply(layered{Name: "base"}, layered{Tags: tagList{"c", "c"}}, policy)

	if !reflect.DeepEqual(got.Tags, tagList{"c"}) {
		t.Errorf("Tags = %v, want the type's dedupe applied even with nothing to merge into", got.Tags)
	}
}

func TestPolicy_MergerWithoutTheMethodPanics(t *testing.T) {
	for name, policy := range map[string]merge.Policy{
		"no Merge method": {Merger: []any{catalog{}}},
		"pointer type":    {Merger: []any{(*tagList)(nil)}},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("no panic, want the policy rejected instead of silently merging structurally")
				}
			}()
			merge.Apply(layered{}, layered{}, policy)
		})
	}
}

func TestPolicy_With_IsTheUnion(t *testing.T) {
	got := merge.Policy{Replace: []any{(*float64)(nil)}, Shared: []any{(*catalog)(nil)}}.
		With(merge.Policy{Replace: []any{(*int)(nil)}, Merger: []any{tagList(nil)}})

	if len(got.Replace) != 2 || len(got.Shared) != 1 || len(got.Merger) != 1 {
		t.Errorf("union = %+v, want both Replace entries, the Shared entry and the Merger entry", got)
	}
}

// Naming a policy type by an untyped nil carries no type information, so it can
// only silently do nothing — fail loud instead.
func TestPolicy_UntypedNilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic for an untyped nil policy entry")
		}
	}()
	merge.Apply(config{}, config{}, merge.Policy{Replace: []any{nil}})
}
