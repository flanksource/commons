package text

import (
	"time"

	"github.com/flanksource/commons/duration"
)

// HumanizeDuration returns a string representing of a duration in the form "3d1h3m".
//
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. Duration more than a day or more than a
// week lose granularity and are truncated to resp. days-hours-minutes and
// weeks-days-hours. The zero duration formats as 0s.
//
// Examples:
//
//	HumanizeDuration(3*time.Hour + 30*time.Minute) // "3h30m"
//	HumanizeDuration(24*time.Hour)                  // "1d"
//	HumanizeDuration(168*time.Hour + 12*time.Hour)  // "1w12h"
func HumanizeDuration(d time.Duration) string {
	return duration.Duration(d).String()
}

// Age returns a human-readable string representing the time elapsed since the given time.
// It's a convenience function that combines time.Since with HumanizeDuration.
//
// Example:
//
//	created := time.Now().Add(-24 * time.Hour)
//	fmt.Printf("Created %s ago", text.Age(created)) // "Created 1d ago"
func Age(d time.Time) string {
	return HumanizeDuration(time.Since(d))
}

// ParseDuration parses a duration string with support for extended units like days and weeks.
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d", "w", "y".
//
// Examples:
//
//	d, _ := ParseDuration("3d12h")    // 3 days 12 hours
//	d, _ := ParseDuration("1w")       // 1 week
//	d, _ := ParseDuration("2h30m")    // 2 hours 30 minutes
func ParseDuration(val string) (*time.Duration, error) {
	d, err := duration.ParseDuration(val)
	if err != nil {
		return nil, err
	}
	t := time.Duration(d)
	return &t, err
}
