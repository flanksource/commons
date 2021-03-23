package text

import (
	"fmt"
	"time"
)

//HumanizeDuration formats a time duration to human readable format
//The duration in floored to the nearest unit. so 75 seconds becomes 1 minute
//biggest unit is days (no weeks, months or years )
func HumanizeDuration(duration time.Duration) string {
	if duration.Seconds() < 2.0 {
		return "1 second"
	} else if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	} else if duration.Minutes() < 2.0 {
		return "1 minute"
	} else if duration.Minutes() < 60.0 {
		return fmt.Sprintf("%d minutes", int64(duration.Minutes()))
	} else if duration.Hours() < 2.0 {
		return "1 hour"
	} else if duration.Hours() < 24.0 {
		return fmt.Sprintf("%d hours", int64(duration.Hours()))
	} else if duration.Hours() < 48.0 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", int64(duration.Hours()/24))
}
