package text

import (
	"time"

	"github.com/flanksource/commons/duration"
)

//  Returns a string representing of a duration in the form "3d1h3m".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. Duration more than a day or more than a
// week lose granularity and are truncated to resp. days-hours-minutes and
// weeks-days-hours. The zero duration formats as 0s.
func HumanizeDuration(d time.Duration) string {
	return duration.Duration(d).String()
}

func ParseDuration(val string) (*time.Duration, error) {
	d, err := duration.ParseDuration(val)
	if err != nil {
		return nil, err
	}
	t := time.Duration(d)
	return &t, err
}
