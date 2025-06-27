package collections

import (
	"reflect"
	"testing"
)

func TestMatchItems(t *testing.T) {
	tests := []struct {
		name     string
		item     string
		patterns []string
		expected bool
	}{
		{
			name:     "Exact Match",
			item:     "apple",
			patterns: []string{"apple"},
			expected: true,
		},
		{
			name:     "Negative Match",
			item:     "apple",
			patterns: []string{"!apple"},
			expected: false,
		},
		{
			name:     "Empty Items List",
			item:     "apple",
			patterns: []string{},
			expected: true,
		},
		{
			name:     "Wildcard Match",
			item:     "apple",
			patterns: []string{"*"},
			expected: true,
		},
		{
			name:     "Wildcard Prefix Match",
			item:     "apple",
			patterns: []string{"appl*"},
			expected: true,
		},
		{
			name:     "Wildcard Suffix Match",
			item:     "apple",
			patterns: []string{"*ple"},
			expected: true,
		},
		{
			name:     "Mixed Matches",
			item:     "apple",
			patterns: []string{"!banana", "appl*", "cherry"},
			expected: true,
		},
		{
			name:     "No Items Match",
			item:     "apple",
			patterns: []string{"!apple", "banana"},
			expected: false,
		},
		{
			name:     "Multiple Wildcards",
			item:     "apple",
			patterns: []string{"ap*e", "*pl*e"},
			expected: false,
		},
		{
			name:     "Glob",
			item:     "apple",
			patterns: []string{"*ppl*"},
			expected: true,
		},
		{
			name:     "Handle whitespaces | should be trimmed",
			item:     "hello",
			patterns: []string{"hello   ", "world"},
			expected: true,
		},
		{
			name:     "Handle whitespaces | should not be trimmed (no match)",
			item:     "hello",
			patterns: []string{"hello%20", "world"},
			expected: false,
		},
		{
			name:     "Handle whitespaces  | should not be trimmed (match)",
			item:     "hello ",
			patterns: []string{"hello%20", "world"},
			expected: true,
		},
		{
			name:     "exclusion and inclusion",
			item:     "mission-control",
			patterns: []string{"!mission-control", "mission-control"},
			expected: false,
		},
		{
			name:     "inclusion and exclusion",
			item:     "mission-control",
			patterns: []string{"mission-control", "!mission-control"},
			expected: false,
		},
		{
			name:     "exclusion",
			item:     "mission-control",
			patterns: []string{"!default"},
			expected: true,
		},
		{
			name:     "Exclude All",
			item:     "anyitem",
			patterns: []string{"!*"},
			expected: false,
		},
		{
			name:     "Exclude All with Inclusion",
			item:     "apple",
			patterns: []string{"!*", "apple"},
			expected: false,
		},
		{
			name:     "Multiple Exclusions",
			item:     "apple",
			patterns: []string{"!banana", "!orange", "!apple"},
			expected: false,
		},
		{
			name:     "Empty Item with Patterns",
			item:     "",
			patterns: []string{"*"},
			expected: true,
		},
		{
			name:     "Empty Pattern String",
			item:     "apple",
			patterns: []string{""},
			expected: false,
		},
		{
			name:     "URL Encoded Pattern Matches",
			item:     "hello ",
			patterns: []string{"hello%20"},
			expected: true,
		},
		{
			name:     "URL Encoded Pattern Does Not Match",
			item:     "hello",
			patterns: []string{"hello%20"},
			expected: false,
		},
		{
			name:     "Malformed URL Encoding",
			item:     "apple",
			patterns: []string{"%zzapple"},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MatchItems(test.item, test.patterns...)
			if result != test.expected {
				t.Errorf("Expected %v but got %v", test.expected, result)
			}
		})
	}
}

func TestAppend(t *testing.T) {
	type args struct {
		slices [][]any
	}
	tests := []struct {
		name string
		args args
		want []any
	}{
		{
			name: "strings",
			args: args{
				slices: [][]any{
					{"a", "b"},
					{"c"},
					{"d", "e", "f"},
				},
			},
			want: []any{"a", "b", "c", "d", "e", "f"},
		},
		{
			name: "ints",
			args: args{
				slices: [][]any{
					{1, 2, 3},
					{4, 5, 6},
					{7, 8, 9},
				},
			},
			want: []any{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Append(tt.args.slices...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Append() = %v, want %v", got, tt.want)
			}
		})
	}
}
