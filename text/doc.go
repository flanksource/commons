// Package text provides utilities for text processing, formatting, and
// manipulation.
//
// The package includes functions for humanizing numbers and durations,
// indenting text, parsing extended duration formats, and safe I/O operations.
//
// Key Features:
//   - Human-readable formatting for bytes, numbers, and durations
//   - Text indentation for hierarchical output
//   - Extended duration parsing (days, weeks, years)
//   - Safe I/O operations that never panic
//
// Byte and Number Formatting:
//
//	// Format bytes in human-readable form
//	text.HumanizeBytes(1536)        // "1.5 KB"
//	text.HumanizeBytes(1048576)     // "1.0 MB"
//	text.HumanizeBytes(5368709120)  // "5.0 GB"
//
//	// Format large numbers with thousand separators
//	text.HumanizeInt(1000000)       // "1,000,000"
//	text.HumanizeInt(42)            // "42"
//
// Duration Formatting:
//
//	// Format durations in human-readable form
//	text.HumanizeDuration(90 * time.Minute)        // "1h30m"
//	text.HumanizeDuration(25 * time.Hour)          // "1d1h"
//	text.HumanizeDuration(8 * 24 * time.Hour)      // "1w1d"
//
//	// Calculate age since a time
//	created := time.Now().Add(-48 * time.Hour)
//	text.Age(created)                               // "2d"
//
// Duration Parsing:
//
// Parse durations with extended units beyond the standard Go time.Duration:
//
//	d, err := text.ParseDuration("3d")       // 72 hours
//	d, err := text.ParseDuration("1w")       // 168 hours
//	d, err := text.ParseDuration("2y")       // ~17520 hours
//	d, err := text.ParseDuration("1d12h30m") // 36.5 hours
//
// Supported units: ns, us/µs, ms, s, m, h, d (days), w (weeks), y (years)
//
// Text Indentation:
//
//	// Indent a string
//	indented := text.String("  ", "line1\nline2\nline3")
//	// "  line1\n  line2\n  line3"
//
//	// Indent bytes
//	data := []byte("line1\nline2")
//	indented := text.Bytes("  ", data)
//
//	// Create an indenting writer
//	writer := text.NewWriter(os.Stdout, "  ")
//	fmt.Fprintln(writer, "This will be indented")
//	fmt.Fprintln(writer, "So will this")
//
// Safe I/O:
//
//	// Read from reader without error handling
//	resp, _ := http.Get("https://example.com")
//	defer resp.Body.Close()
//	content := text.SafeRead(resp.Body) // Empty string on error
//
// Formatting Examples:
//
//	// Display file sizes
//	fmt.Printf("Size: %s\n", text.HumanizeBytes(fileSize))
//
//	// Display request counts
//	fmt.Printf("Requests: %s\n", text.HumanizeInt(requestCount))
//
//	// Display uptime
//	fmt.Printf("Uptime: %s\n", text.HumanizeDuration(time.Since(startTime)))
//
//	// Display last updated
//	fmt.Printf("Updated: %s ago\n", text.Age(lastModified))
//
// Related Packages:
//   - duration: Extended duration parsing (used internally)
package text
