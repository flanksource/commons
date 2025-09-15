package collections

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MatchItems", func() {
	DescribeTable("pattern matching scenarios",
		func(item string, patterns []string, expected bool) {
			result := MatchItems(item, patterns...)
			Expect(result).To(Equal(expected))
		},
		Entry("exact match", "apple", []string{"apple"}, true),
		Entry("negative match", "apple", []string{"!apple"}, false),
		Entry("empty items list", "apple", []string{}, true),
		Entry("wildcard match", "apple", []string{"*"}, true),
		Entry("wildcard prefix match", "apple", []string{"appl*"}, true),
		Entry("wildcard suffix match", "apple", []string{"*ple"}, true),
		Entry("mixed matches", "apple", []string{"!banana", "appl*", "cherry"}, true),
		Entry("no items match", "apple", []string{"!apple", "banana"}, false),
		Entry("multiple wildcards", "apple", []string{"ap*e", "*pl*e"}, false),
		Entry("glob", "apple", []string{"*ppl*"}, true),
		Entry("handle whitespaces - should be trimmed", "hello", []string{"hello   ", "world"}, true),
		Entry("handle whitespaces - should not be trimmed (no match)", "hello", []string{"hello%20", "world"}, false),
		Entry("handle whitespaces - should not be trimmed (match)", "hello ", []string{"hello%20", "world"}, true),
		Entry("exclusion and inclusion", "mission-control", []string{"!mission-control", "mission-control"}, false),
		Entry("inclusion and exclusion", "mission-control", []string{"mission-control", "!mission-control"}, false),
		Entry("exclusion", "mission-control", []string{"!default"}, true),
		Entry("exclude all", "anyitem", []string{"!*"}, false),
		Entry("exclude all with inclusion", "apple", []string{"!*", "apple"}, false),
		Entry("multiple exclusions", "apple", []string{"!banana", "!orange", "!apple"}, false),
		Entry("empty item with patterns", "", []string{"*"}, true),
		Entry("empty pattern string", "apple", []string{""}, false),
		Entry("URL encoded pattern matches", "hello ", []string{"hello%20"}, true),
		Entry("URL encoded pattern does not match", "hello", []string{"hello%20"}, false),
		Entry("malformed URL encoding", "apple", []string{"%zzapple"}, false),
	)
})

var _ = Describe("Append", func() {
	It("should append string slices", func() {
		slices := [][]any{
			{"a", "b"},
			{"c"},
			{"d", "e", "f"},
		}

		result := Append(slices...)

		Expect(result).To(Equal([]any{"a", "b", "c", "d", "e", "f"}))
	})

	It("should append integer slices", func() {
		slices := [][]any{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		}

		result := Append(slices...)

		Expect(result).To(Equal([]any{1, 2, 3, 4, 5, 6, 7, 8, 9}))
	})
})
