package merge_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/flanksource/commons/merge"
)

type strategicItem struct {
	Name string `json:"name"`
	A    string `json:"a,omitempty"`
	B    string `json:"b,omitempty"`
}

type strategicGroup struct {
	Name    string          `json:"name"`
	Members []strategicItem `json:"members,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

// strategicHolder declares its lists the way Kubernetes API types do, so they
// merge the way `kubectl patch` merges them.
type strategicHolder struct {
	Items  []strategicItem   `json:"items" patchStrategy:"merge" patchMergeKey:"name"`
	Groups []strategicGroup  `json:"groups,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Tags   []string          `json:"tags,omitempty" patchStrategy:"merge"`
	Plain  []strategicItem   `json:"plain,omitempty"`
	Nested *strategicHolder  `json:"nested,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

func TestApply_PatchStrategyMergesElementsSharingAKey(t *testing.T) {
	base := strategicHolder{Items: []strategicItem{{Name: "x", A: "base-a", B: "base-b"}}}
	override := strategicHolder{Items: []strategicItem{{Name: "x", A: "override-a"}}}

	got := merge.Apply(base, override, merge.Policy{})

	want := []strategicItem{{Name: "x", A: "override-a", B: "base-b"}}
	if !reflect.DeepEqual(got.Items, want) {
		t.Errorf("Items = %+v, want the override's fields over the base's, unset ones kept: %+v", got.Items, want)
	}
	if !reflect.DeepEqual(base.Items, []strategicItem{{Name: "x", A: "base-a", B: "base-b"}}) {
		t.Errorf("base mutated through the merge: %+v", base.Items)
	}
}

// Kubernetes orders the result by the patch first: elements the override names
// come in the override's order, and a base-only element is placed after the
// named elements it cannot be ordered against. The expectations below follow
// that rule rather than a concatenation.
func TestApply_PatchStrategyAddsNewKeysAndKeepsUnmentionedOnes(t *testing.T) {
	base := strategicHolder{Items: []strategicItem{{Name: "x", A: "x-a"}, {Name: "y", A: "y-a"}}}
	override := strategicHolder{Items: []strategicItem{{Name: "x", B: "x-b"}, {Name: "z", B: "z-b"}}}

	got := merge.Apply(base, override, merge.Policy{})

	want := []strategicItem{{Name: "x", A: "x-a", B: "x-b"}, {Name: "z", B: "z-b"}, {Name: "y", A: "y-a"}}
	if !reflect.DeepEqual(got.Items, want) {
		t.Errorf("Items = %+v, want x merged, z added and y kept: %+v", got.Items, want)
	}
}

func TestApply_PatchStrategyMergesNestedTaggedLists(t *testing.T) {
	base := strategicHolder{Groups: []strategicGroup{{Name: "g", Members: []strategicItem{{Name: "m", A: "base-a"}}}}}
	override := strategicHolder{Groups: []strategicGroup{{Name: "g", Members: []strategicItem{{Name: "m", B: "override-b"}}}}}

	got := merge.Apply(base, override, merge.Policy{})

	want := []strategicGroup{{Name: "g", Members: []strategicItem{{Name: "m", A: "base-a", B: "override-b"}}}}
	if !reflect.DeepEqual(got.Groups, want) {
		t.Errorf("Groups = %+v, want the element type's own patch tags honoured: %+v", got.Groups, want)
	}
}

func TestApply_PatchStrategyOnScalarsIsAUnion(t *testing.T) {
	got := merge.Apply(strategicHolder{Tags: []string{"a", "b"}}, strategicHolder{Tags: []string{"b", "c"}}, merge.Policy{})

	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got.Tags, want) {
		t.Errorf("Tags = %v, want the set union %v", got.Tags, want)
	}
}

func TestApply_PatchStrategyIsReadThroughAPointer(t *testing.T) {
	base := strategicHolder{Nested: &strategicHolder{Items: []strategicItem{{Name: "x", A: "base-a"}}}}
	override := strategicHolder{Nested: &strategicHolder{Items: []strategicItem{{Name: "x", B: "override-b"}}}}

	got := merge.Apply(base, override, merge.Policy{})

	if want := []strategicItem{{Name: "x", A: "base-a", B: "override-b"}}; !reflect.DeepEqual(got.Nested.Items, want) {
		t.Errorf("Nested.Items = %+v, want the tags honoured behind a pointer too: %+v", got.Nested.Items, want)
	}
}

func TestApply_PatchStrategyWithOneSideEmpty(t *testing.T) {
	baseOnly := strategicHolder{Items: []strategicItem{{Name: "x", A: "base-a"}}}
	if got := merge.Apply(baseOnly, strategicHolder{}, merge.Policy{}); !reflect.DeepEqual(got.Items, baseOnly.Items) {
		t.Errorf("Items = %+v, want an empty override to leave the base alone", got.Items)
	}
	overrideOnly := strategicHolder{Items: []strategicItem{{Name: "z", B: "override-b"}}}
	if got := merge.Apply(strategicHolder{}, overrideOnly, merge.Policy{}); !reflect.DeepEqual(got.Items, overrideOnly.Items) {
		t.Errorf("Items = %+v, want the override to reach a base that had none", got.Items)
	}
}

func TestApply_UntaggedSliceBesidePatchStrategyIsReplaced(t *testing.T) {
	base := strategicHolder{Plain: []strategicItem{{Name: "x", A: "base-a"}}}
	override := strategicHolder{Plain: []strategicItem{{Name: "x", B: "override-b"}}}

	got := merge.Apply(base, override, merge.Policy{})

	if !reflect.DeepEqual(got.Plain, override.Plain) {
		t.Errorf("Plain = %+v, want an untagged slice still replaced wholesale", got.Plain)
	}
}

type undeclaredMergeKey struct {
	Items []strategicItem `json:"items" patchStrategy:"merge" patchMergeKey:"id"`
}

type patchStrategyAndAppend struct {
	Items []strategicItem `json:"items" merge:"append" patchStrategy:"merge" patchMergeKey:"name"`
}

type patchStrategyOnAScalar struct {
	Name string `json:"name" patchStrategy:"merge"`
}

type mergeKeyWithoutStrategy struct {
	Items []strategicItem `json:"items" patchMergeKey:"name"`
}

// A patch declaration that cannot be honoured fails at the merge, naming the
// field, rather than degrading to wholesale replacement.
func TestApply_UnsatisfiablePatchStrategyPanics(t *testing.T) {
	for name, c := range map[string]struct {
		field string
		apply func()
	}{
		"an element without the declared merge key": {
			field: "undeclaredMergeKey.Items",
			apply: func() {
				merge.Apply(
					undeclaredMergeKey{Items: []strategicItem{{Name: "x"}}},
					undeclaredMergeKey{Items: []strategicItem{{Name: "y"}}},
					merge.Policy{})
			},
		},
		"both a merge rule and a patch strategy": {
			field: "patchStrategyAndAppend.Items",
			apply: func() { merge.Apply(patchStrategyAndAppend{}, patchStrategyAndAppend{}, merge.Policy{}) },
		},
		"a patch strategy on something that is not a list": {
			field: "patchStrategyOnAScalar.Name",
			apply: func() { merge.Apply(patchStrategyOnAScalar{}, patchStrategyOnAScalar{}, merge.Policy{}) },
		},
		"a merge key without a patch strategy": {
			field: "mergeKeyWithoutStrategy.Items",
			apply: func() { merge.Apply(mergeKeyWithoutStrategy{}, mergeKeyWithoutStrategy{}, merge.Policy{}) },
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("no panic, want the declaration rejected instead of silently replaced")
				}
				if message := fmt.Sprint(recovered); !strings.Contains(message, c.field) {
					t.Errorf("panic %q does not name the field %s", message, c.field)
				}
			}()
			c.apply()
		})
	}
}
