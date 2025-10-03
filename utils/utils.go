// Package utils provides common utility functions for Go applications.
//
// The package includes helpers for pointer operations, string manipulation,
// randomization, template interpolation, time parsing, and named locks. These
// utilities address common patterns in Go programming and reduce boilerplate code.
//
// Key features:
//   - Generic pointer and dereference utilities
//   - Coalesce function for finding first non-zero value
//   - Environment variable helpers
//   - Random string and key generation
//   - Template string interpolation
//   - Flexible time parsing with multiple formats
//   - Named lock implementation for synchronization
//
// Pointer utilities:
//
//	// Create pointer to value
//	strPtr := utils.Ptr("hello")
//	intPtr := utils.Ptr(42)
//
//	// Safely dereference (returns zero value if nil)
//	val := utils.Deref(strPtr) // "hello"
//	val2 := utils.Deref(nil)   // "" (zero value)
//
// Coalesce example:
//
//	// Returns first non-zero value
//	result := utils.Coalesce("", "", "value", "ignored") // "value"
//	port := utils.Coalesce(0, 0, 8080, 9090)            // 8080
//
// Random generation:
//
//	// Generate random hex key
//	apiKey := utils.RandomKey(32)
//
//	// Generate random alphanumeric string
//	sessionID := utils.RandomString(16)
//
// Template interpolation:
//
//	vars := map[string]string{"name": "World", "time": "today"}
//	result := utils.Interpolate("Hello {{.name}}, how are you {{.time}}?", vars)
//	// "Hello World, how are you today?"
//
// Time parsing:
//
//	// Parse time with automatic format detection
//	t := utils.ParseTime("2023-12-25T10:30:00Z")        // RFC3339
//	t2 := utils.ParseTime("2023-12-25 10:30:00")       // MySQL format
//	t3 := utils.ParseTime("2023-12-25")                // Date only
//
// Named locks:
//
//	lock := &utils.NamedLock{}
//	if unlocker := lock.TryLock("resource-1", 5*time.Second); unlocker != nil {
//		defer unlocker.Release()
//		// Critical section
//	}
package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Ptr returns a pointer to the given value.
// This is a generic helper function useful for creating pointers to literals
// or values that need to be passed as pointers to functions.
//
// Example:
//
//	// Instead of:
//	temp := "hello"
//	ptr := &temp
//
//	// You can write:
//	ptr := utils.Ptr("hello")
func Ptr[T any](value T) *T {
	return &value
}

// Deref safely dereferences a pointer, returning the zero value if the pointer is nil.
// This prevents nil pointer panics and simplifies nil checking.
//
// Example:
//
//	var strPtr *string = nil
//	val := utils.Deref(strPtr)  // Returns "" (zero value for string)
//
//	strPtr = utils.Ptr("hello")
//	val = utils.Deref(strPtr)   // Returns "hello"
func Deref[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}

	return *v
}

// Coalesce returns the first non-zero value from the provided arguments.
// This is similar to the COALESCE function in SQL and the nullish coalescing
// operator (??) in other languages.
//
// Example:
//
//	// String coalescing
//	name := utils.Coalesce("", "", "John", "Jane") // Returns "John"
//
//	// Number coalescing
//	port := utils.Coalesce(0, 0, 8080, 9090)      // Returns 8080
//
//	// With variables
//	result := utils.Coalesce(config.URL, os.Getenv("API_URL"), "http://localhost")
func Coalesce[T comparable](arr ...T) T {
	var zeroVal T
	for _, item := range arr {
		if item != zeroVal {
			return item
		}
	}

	return zeroVal
}

// GetEnvOrDefault returns the value of the first non-empty environment variable
// from the provided list of names. This is useful for checking multiple possible
// environment variable names or providing fallback options.
//
// Example:
//
//	// Check multiple possible names
//	dbHost := utils.GetEnvOrDefault("DATABASE_HOST", "DB_HOST", "POSTGRES_HOST")
//
//	// With fallback handling
//	apiKey := utils.GetEnvOrDefault("API_KEY", "SECRET_KEY")
//	if apiKey == "" {
//		apiKey = "default-key"
//	}
func GetEnvOrDefault(names ...string) string {
	for _, name := range names {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	return ""
}

// ShortTimestamp returns a shortened timestamp using
// week of year + day of week to represent a day of the
// e.g. 1st of Jan on a Tuesday is 13
func ShortTimestamp() string {
	_, week := time.Now().ISOWeek()
	return fmt.Sprintf("%d%d-%s", week, time.Now().Weekday(), time.Now().Format("150405"))
}

func RandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Cannot generate random data: %v", err)
		return ""
	}
	return hex.EncodeToString(bytes)
}

// randomChars defines the alphanumeric characters that can be part of a random string
const randomChars = "0123456789abcdefghijklmnopqrstuvwxyz"

// RandomString returns a random string consisting of the characters in
// randomChars, with the length customized by the parameter
func RandomString(length int) string {
	// len("0123456789abcdefghijklmnopqrstuvwxyz") = 36 which doesn't evenly divide
	// the possible values of a byte: 256 mod 36 = 4. Discard any random bytes we
	// read that are >= 252 so the bytes we evenly divide the character set.
	const maxByteValue = 252

	var (
		b     byte
		err   error
		token = make([]byte, length)
	)

	reader := bufio.NewReaderSize(rand.Reader, length*2)
	for i := range token {
		for {
			if b, err = reader.ReadByte(); err != nil {
				return ""
			}
			if b < maxByteValue {
				break
			}
		}

		token[i] = randomChars[int(b)%len(randomChars)]
	}

	return string(token)
}

// Interpolate templatises the string using the vars as the context
func Interpolate(arg string, vars interface{}) string {
	tmpl, err := template.New("test").Parse(arg)
	if err != nil {
		log.Errorf("Failed to parse template %s -> %s\n", arg, err)
		return arg
	}
	buf := bytes.NewBufferString("")

	err = tmpl.Execute(buf, vars)
	if err != nil {
		log.Errorf("Failed to execute template %s -> %s\n", arg, err)
		return arg
	}
	return buf.String()

}

// InterpolateStrings templatises each string in the slice using the vars as the context
func InterpolateStrings(arg []string, vars interface{}) []string {
	out := make([]string, len(arg))
	for i, e := range arg {
		out[i] = Interpolate(e, vars)
	}
	return out
}

// NormalizeVersion appends "v" to version string if it's not exist
func NormalizeVersion(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// Stringify converts the given value to a string.
// If the value is already a string, it is returned as is.
func Stringify(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
