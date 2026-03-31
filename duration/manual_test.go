package duration

import (
	"fmt"
	"testing"
	"time"
)

func TestMillisecondFormatting(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{320 * time.Millisecond, "320ms"},
		{150 * time.Millisecond, "150ms"},
		{5250 * time.Millisecond, "5.25s"},
		{1250 * time.Millisecond, "1.25s"},
		{999 * time.Millisecond, "999ms"},
		{1001 * time.Millisecond, "1.001s"},
		{50 * time.Millisecond, "50ms"},
	}
	for _, tc := range tests {
		got := Duration(tc.d).String()
		if got != tc.expected {
			t.Errorf("%v: got %q, want %q", tc.d, got, tc.expected)
		}
		fmt.Printf("%v -> %s (want %s)\n", tc.d, got, tc.expected)
	}
}
