package collections

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeMap", func() {
	It("should merge maps with no overlaps", func() {
		a := map[string]string{"name": "flanksource"}
		b := map[string]string{"foo": "bar"}
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should merge maps with overlaps, b takes precedence", func() {
		a := map[string]string{"name": "flanksource", "foo": "baz"}
		b := map[string]string{"foo": "bar"}
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should merge maps with multiple overlaps", func() {
		a := map[string]string{"name": "github", "foo": "baz"}
		b := map[string]string{"name": "flanksource", "foo": "bar"}
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should handle identical maps", func() {
		a := map[string]string{"name": "flanksource", "foo": "bar"}
		b := map[string]string{"name": "flanksource", "foo": "bar"}
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should handle nil first map", func() {
		var a map[string]string
		b := map[string]string{"name": "flanksource", "foo": "bar"}
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should handle nil second map", func() {
		a := map[string]string{"name": "flanksource", "foo": "bar"}
		var b map[string]string
		expected := map[string]string{
			"name": "flanksource",
			"foo":  "bar",
		}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})

	It("should handle both maps nil", func() {
		var a, b map[string]string
		expected := map[string]string{}

		result := MergeMap(a, b)

		Expect(result).To(Equal(expected))
	})
})

var _ = Describe("KeyValueSliceToMap", func() {
	It("should convert simple key=value pair", func() {
		args := []string{"name=flanksource"}
		expected := map[string]string{"name": "flanksource"}

		result := KeyValueSliceToMap(args)

		Expect(result).To(Equal(expected))
	})

	It("should handle whitespace around key=value pairs", func() {
		args := []string{"    name  =  flanksource   "}
		expected := map[string]string{"name": "flanksource"}

		result := KeyValueSliceToMap(args)

		Expect(result).To(Equal(expected))
	})

	It("should convert multiple key=value pairs", func() {
		args := []string{"name=flanksource", "foo=bar"}
		expected := map[string]string{"name": "flanksource", "foo": "bar"}

		result := KeyValueSliceToMap(args)

		Expect(result).To(Equal(expected))
	})

	It("should handle values with equal signs", func() {
		args := []string{"name=foo=bar"}
		expected := map[string]string{"name": "foo=bar"}

		result := KeyValueSliceToMap(args)

		Expect(result).To(Equal(expected))
	})
})
