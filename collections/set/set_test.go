package set

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StringSet(t *testing.T) {
	s := New("a", "b", "c", "d")
	s.Add("e")

	assert.ElementsMatch(t, s.ToSlice(), []string{"a", "b", "c", "d", "e"})

	s.Add("a")
	assert.ElementsMatch(t, s.ToSlice(), []string{"a", "b", "c", "d", "e"})

	s.Remove("b")
	s.Remove("c")
	assert.ElementsMatch(t, s.ToSlice(), []string{"a", "d", "e"})

	assert.Equal(t, s.Contains("a"), true)
	assert.Equal(t, s.Contains("z"), false)

	s2 := New("d", "e", "f", "g", "h")
	assert.ElementsMatch(t, s.Union(s2).ToSlice(), []string{"a", "d", "e", "f", "g", "h"})

	assert.ElementsMatch(t, s.Intersection(s2).ToSlice(), []string{"d", "e"})
}

func Test_IntSet(t *testing.T) {
	s := New(1, 2, 3, 4)
	s.Add(5)

	assert.ElementsMatch(t, s.ToSlice(), []int{1, 2, 3, 4, 5})

	s.Add(1)
	assert.ElementsMatch(t, s.ToSlice(), []int{1, 2, 3, 4, 5})

	s.Remove(2)
	s.Remove(3)
	assert.ElementsMatch(t, s.ToSlice(), []int{1, 4, 5})

	assert.Equal(t, s.Contains(1), true)
	assert.Equal(t, s.Contains(100), false)

	s2 := New(4, 5, 6, 7, 8)
	assert.ElementsMatch(t, s.Union(s2).ToSlice(), []int{1, 4, 5, 6, 7, 8})

	assert.ElementsMatch(t, s.Intersection(s2).ToSlice(), []int{4, 5})
}

func Test_JSON(t *testing.T) {
	type Fruits struct {
		Names Set[string] `json:"names"`
	}

	f := Fruits{Names: New("orange", "apple", "orange", "banana", "mango")}

	b, err := json.Marshal(f)
	assert.NoError(t, err)

	assert.Equal(t, len(`{"names":["banana","mango","orange","apple"]}`), len(string(b)))

	var jsonFruits Fruits
	err = json.Unmarshal(b, &jsonFruits)
	assert.NoError(t, err)

	assert.ElementsMatch(t, f.Names.ToSlice(), jsonFruits.Names.ToSlice())
}
