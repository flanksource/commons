package utils

import "time"

// ParseTime attempts to parse a time string using multiple common formats.
// It tries various formats in order and returns the first successful parse.
// Returns nil if no format matches.
//
// Supported formats:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - RFC3339Nano: "2006-01-02T15:04:05.999999999Z07:00"
//   - ANSIC: "Mon Jan _2 15:04:05 2006"
//   - DateTime: "2006-01-02 15:04:05"
//   - DateOnly: "2006-01-02"
//   - ISO8601 without timezone: "2006-01-02T15:04:05"
//   - MySQL datetime: "2006-01-02 15:04:05"
//
// Example:
//
//	t1 := utils.ParseTime("2023-12-25T10:30:00Z")      // RFC3339
//	t2 := utils.ParseTime("2023-12-25 10:30:00")       // MySQL format
//	t3 := utils.ParseTime("2023-12-25")                // Date only
//	t4 := utils.ParseTime("Mon Jan 2 15:04:05 2006")   // ANSIC
//	
//	if t := utils.ParseTime(userInput); t != nil {
//		fmt.Printf("Parsed time: %v\n", t)
//	} else {
//		fmt.Println("Invalid time format")
//	}
func ParseTime(t string) *time.Time {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.ANSIC,
		time.DateTime,
		time.DateOnly,
		"2006-01-02T15:04:05", // ISO8601 without timezone
		"2006-01-02 15:04:05", // MySQL datetime format
	}

	for _, format := range formats {
		parsed, err := time.Parse(format, t)
		if err == nil {
			return &parsed
		}
	}

	return nil
}
