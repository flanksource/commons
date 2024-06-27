package collections

import (
	"reflect"
	"testing"
)

func TestMatchItems(t *testing.T) {
	tests := []struct {
		name     string
		item     string
		items    []string
		expected bool
	}{
		{
			name:     "Exact Match",
			item:     "apple",
			items:    []string{"apple"},
			expected: true,
		},
		{
			name:     "Negative Match",
			item:     "apple",
			items:    []string{"!apple"},
			expected: false,
		},
		{
			name:     "Empty Items List",
			item:     "apple",
			items:    []string{},
			expected: true,
		},
		{
			name:     "Wildcard Match",
			item:     "apple",
			items:    []string{"*"},
			expected: true,
		},
		{
			name:     "Wildcard Prefix Match",
			item:     "apple",
			items:    []string{"appl*"},
			expected: true,
		},
		{
			name:     "Wildcard Suffix Match",
			item:     "apple",
			items:    []string{"*ple"},
			expected: true,
		},
		{
			name:     "Mixed Matches",
			item:     "apple",
			items:    []string{"!banana", "appl*", "cherry"},
			expected: true,
		},
		{
			name:     "No Items Match",
			item:     "apple",
			items:    []string{"!apple", "banana"},
			expected: false,
		},
		{
			name:     "Multiple Wildcards",
			item:     "apple",
			items:    []string{"ap*e", "*p*"},
			expected: false,
		},
		{
			name:     "Handle whitespaces | should be trimmed",
			item:     "hello",
			items:    []string{"hello   ", "world"},
			expected: true,
		},
		{
			name:     "Handle whitespaces | should not be trimmed (no match)",
			item:     "hello",
			items:    []string{"hello%20", "world"},
			expected: false,
		},
		{
			name:     "Handle whitespaces  | should not be trimmed (match)",
			item:     "hello ",
			items:    []string{"hello%20", "world"},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MatchItems(test.item, test.items...)
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
