package utils

import "time"

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
