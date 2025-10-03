// Package is provides simple type checking and environment detection utilities.
//
// The package offers concise boolean functions for common type checks and
// runtime environment detection. All functions follow the pattern is.X()
// for readable, natural-language code.
//
// Key Features:
//   - Type checking with reflection
//   - File and archive detection
//   - Terminal/TTY detection
//   - Simple, readable API
//
// Type Checking:
//
//	// Check if value is a slice
//	data := []string{"a", "b", "c"}
//	if is.Slice(data) {
//		// data is a slice
//	}
//
//	// Works with any type
//	numbers := []int{1, 2, 3}
//	is.Slice(numbers)  // true
//
//	str := "hello"
//	is.Slice(str)      // false
//
// File Detection:
//
//	// Check if file exists
//	if is.File("/etc/hosts") {
//		// File exists
//	}
//
//	// Check if file is an archive
//	if is.Archive("package.tar.gz") {
//		// Extract the archive
//	}
//
//	// Supported archive formats
//	is.Archive("file.zip")     // true
//	is.Archive("file.tar.gz")  // true
//	is.Archive("file.gz")      // true
//	is.Archive("file.xz")      // true
//	is.Archive("file.txz")     // true
//	is.Archive("file.txt")     // false
//
// Terminal Detection:
//
//	// Check if running in an interactive terminal
//	if is.TTY() {
//		// Enable colored output, progress bars, etc.
//		fmt.Println("\033[32mGreen text\033[0m")
//	} else {
//		// Plain output for logs, pipes, redirects
//		fmt.Println("Plain text")
//	}
//
// Common Use Cases:
//
//	// Conditional output formatting
//	if is.TTY() {
//		showProgressBar()
//	} else {
//		logToFile()
//	}
//
//	// File processing
//	if is.File(configPath) {
//		loadConfig(configPath)
//	}
//
//	// Archive handling
//	if is.Archive(input) {
//		extractArchive(input)
//	} else {
//		processFile(input)
//	}
//
//	// Generic type checking
//	if is.Slice(value) {
//		processSlice(value)
//	}
//
// Limitations:
//
// The Slice function uses reflection and only checks if a value is a slice type.
// It doesn't validate slice element types or other slice properties.
//
// The Archive function only checks file extensions, not actual file content.
// For robust archive detection, consider inspecting file headers/magic numbers.
package is
