package text

import (
	"io"
)

// SafeRead reads all data from an io.Reader and returns it as a string.
// If an error occurs during reading, it returns an empty string instead of
// propagating the error. This is useful when you want to ensure a read
// operation never fails, such as in logging or display contexts.
//
// Example:
//
//	resp, _ := http.Get("https://example.com")
//	defer resp.Body.Close()
//	content := text.SafeRead(resp.Body)
//	// content will be empty string if read fails
func SafeRead(r io.Reader) string {
	data, _ := io.ReadAll(r)
	return string(data)
}
