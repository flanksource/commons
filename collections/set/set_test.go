package set

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Set", func() {
	Describe("String Set", func() {
		It("should handle basic string set operations", func() {
			s := New("a", "b", "c", "d")
			s.Add("e")

			Expect(s.ToSlice()).To(ConsistOf("a", "b", "c", "d", "e"))

			s.Add("a") // Adding duplicate
			Expect(s.ToSlice()).To(ConsistOf("a", "b", "c", "d", "e"))

			s.Remove("b")
			s.Remove("c")
			Expect(s.ToSlice()).To(ConsistOf("a", "d", "e"))

			Expect(s.Contains("a")).To(BeTrue())
			Expect(s.Contains("z")).To(BeFalse())
		})

		It("should handle union operations", func() {
			s := New("a", "d", "e")
			s2 := New("d", "e", "f", "g", "h")

			result := s.Union(s2)
			Expect(result.ToSlice()).To(ConsistOf("a", "d", "e", "f", "g", "h"))
		})

		It("should handle intersection operations", func() {
			s := New("a", "d", "e")
			s2 := New("d", "e", "f", "g", "h")

			result := s.Intersection(s2)
			Expect(result.ToSlice()).To(ConsistOf("d", "e"))
		})
	})

	Describe("Integer Set", func() {
		It("should handle basic integer set operations", func() {
			s := New(1, 2, 3, 4)
			s.Add(5)

			Expect(s.ToSlice()).To(ConsistOf(1, 2, 3, 4, 5))

			s.Add(1) // Adding duplicate
			Expect(s.ToSlice()).To(ConsistOf(1, 2, 3, 4, 5))

			s.Remove(2)
			s.Remove(3)
			Expect(s.ToSlice()).To(ConsistOf(1, 4, 5))

			Expect(s.Contains(1)).To(BeTrue())
			Expect(s.Contains(100)).To(BeFalse())
		})

		It("should handle union operations", func() {
			s := New(1, 4, 5)
			s2 := New(4, 5, 6, 7, 8)

			result := s.Union(s2)
			Expect(result.ToSlice()).To(ConsistOf(1, 4, 5, 6, 7, 8))
		})

		It("should handle intersection operations", func() {
			s := New(1, 4, 5)
			s2 := New(4, 5, 6, 7, 8)

			result := s.Intersection(s2)
			Expect(result.ToSlice()).To(ConsistOf(4, 5))
		})
	})

	Describe("JSON Serialization", func() {
		It("should marshal and unmarshal sets correctly", func() {
			type Fruits struct {
				Names Set[string] `json:"names"`
			}

			f := Fruits{Names: New("orange", "apple", "orange", "banana", "mango")}

			b, err := json.Marshal(f)
			Expect(err).ToNot(HaveOccurred())

			// Check that the JSON has the expected length (order may vary)
			Expect(len(string(b))).To(Equal(len(`{"names":["banana","mango","orange","apple"]}`)))

			var jsonFruits Fruits
			err = json.Unmarshal(b, &jsonFruits)
			Expect(err).ToNot(HaveOccurred())

			Expect(jsonFruits.Names.ToSlice()).To(ConsistOf(f.Names.ToSlice()))
		})
	})
})
