package json

import (
	"reflect"
	"testing"
)

// TestUnmarshalPreservesInts pins the behaviour of the sigs.k8s.io/json decoder
// this package replaces: integral numbers decode to int64, everything else to
// float64, at any depth.
func TestUnmarshalPreservesInts(t *testing.T) {
	input := []byte(`{"int":9223372036854775807,"float":1.5,"exp":1e3,"nested":[{"n":-2}]}`)
	want := map[string]interface{}{
		"int":    int64(9223372036854775807),
		"float":  1.5,
		"exp":    float64(1000),
		"nested": []interface{}{map[string]interface{}{"n": int64(-2)}},
	}

	got := map[string]interface{}{}
	if err := Unmarshal(input, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal = %#v, want %#v", got, want)
	}
}

func TestUnmarshalUntypedTargets(t *testing.T) {
	var list []interface{}
	if err := Unmarshal([]byte(`[1,"a"]`), &list); err != nil {
		t.Fatalf("Unmarshal into *[]interface{}: %v", err)
	}
	if want := []interface{}{int64(1), "a"}; !reflect.DeepEqual(list, want) {
		t.Errorf("list = %#v, want %#v", list, want)
	}

	var value interface{}
	if err := Unmarshal([]byte(`7`), &value); err != nil {
		t.Fatalf("Unmarshal into *interface{}: %v", err)
	}
	if value != int64(7) {
		t.Errorf("value = %#v, want int64(7)", value)
	}
}

func TestUnmarshalRejects(t *testing.T) {
	cases := map[string]struct {
		input  string
		target interface{}
	}{
		"not json":            {input: `<THIS IS NOT JSON>`, target: &map[string]interface{}{}},
		"trailing data":       {input: `{} {}`, target: &map[string]interface{}{}},
		"float overflow":      {input: `{"f":1e400}`, target: &map[string]interface{}{}},
		"case-folding struct": {input: `{"name":"x"}`, target: &struct{ Name string }{}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Unmarshal([]byte(c.input), c.target); err == nil {
				t.Errorf("Unmarshal(%s) into %T succeeded, want an error", c.input, c.target)
			}
		})
	}
}
