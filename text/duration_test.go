package text

import (
	"testing"
	"time"
)

func TestHumanizeDuration(t *testing.T) {

	tests := []struct {
		Duration  time.Duration
		Humanized string
	}{
		{5 * time.Second, "5 seconds"},
		{75 * time.Second, "1 minute"},
		{121 * time.Second, "2 minutes"},
		{431 * time.Second, "7 minutes"},
		{65 * time.Minute, "1 hour"},
		{125 * time.Minute, "2 hours"},
		{23 * time.Hour, "23 hours"},
		{32 * time.Hour, "1 day"},
		{49 * time.Hour, "2 days"},
		{320 * time.Hour, "13 days"},
	}

	for _, tc := range tests {
		if HumanizeDuration(tc.Duration) != tc.Humanized {
			t.Errorf("Failed for test case %v ", tc)
		}
	}
}
