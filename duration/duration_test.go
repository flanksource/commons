package duration

import (
	"testing"
	"time"
)

func TestHumanizeDuration(t *testing.T) {
	tests := []struct {
		Duration  time.Duration
		Humanized string
	}{
		{5 * time.Nanosecond, "5ns"},
		{5 * time.Millisecond, "5ms"},
		{1250 * time.Millisecond, "1.25s"},
		{121*time.Second + 250*time.Millisecond, "2m1s"},

		{5 * time.Second, "5s"},
		{75*time.Second + 250*time.Millisecond, "1m15.25s"},
		{121*time.Second + 250*time.Millisecond, "2m1s"},
		{431 * time.Second, "7m11s"},
		{65 * time.Minute, "1h5m"},
		{125 * time.Minute, "2h5m"},
		{23 * time.Hour, "23h"},
		{32 * time.Hour, "1d8h"},
		{49 * time.Hour, "2d1h"},
		{320 * time.Hour, "1w6d8h"},
		{3200 * time.Hour, "19w0d8h"},
	}

	for _, tc := range tests {
		if Duration(tc.Duration).String() != tc.Humanized {
			t.Errorf("Failed for test case %v != %v", tc, Duration(tc.Duration).String())
		}
	}
}
