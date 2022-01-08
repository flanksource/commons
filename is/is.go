package is

import (
	"os"
	"reflect"
	"strings"
)

// Slice returns true if the argument is a slice
func Slice(arg interface{}) bool {
	return reflect.ValueOf(arg).Kind() == reflect.Slice
}

// TTY returns true if running inside an interactive terminal
func TTY() bool {
	fi, _ := os.Stdout.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// File returns if the file exists
func File(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Archive returns if the file represents an archive that files understands
func Archive(filename string) bool {
	return strings.HasSuffix(filename, ".zip") ||
		strings.HasSuffix(filename, ".tar.gz") ||
		strings.HasSuffix(filename, ".gz") ||
		strings.HasSuffix(filename, ".xz")
}
