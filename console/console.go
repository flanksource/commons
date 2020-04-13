package console

import (
	"fmt"

	"github.com/flanksource/commons/is"
)

var (
	reset        = "\x1b[0m"
	red          = "\x1b[31m"
	yellow       = "\x1b[33m"
	gray         = "\x1b[37m"
	lightRed     = "\x1b[31;1m"
	green        = "\x1b[32m"
	lightGreen   = "\x1b[32;1m"
	lightBlue    = "\x1b[34;1m"
	magenta      = "\x1b[35m"
	lightMagenta = "\x1b[35;1m"
	cyan         = "\x1b[36m"
	lightCyan    = "\x1b[36;1m"
	white        = "\x1b[37;1m"
	bold         = "\x1b[1m"
	boldOff      = "\x1b[22m"
	darkWhite    = "\x1b[38;5;244m"
)

// DarkWhitef prints and formats msg as dark white
func DarkF(msg string, args ...interface{}) string {
	if is.TTY() {
		return darkWhite + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Redf prints and formats msg as red text
func Redf(msg string, args ...interface{}) string {
	if is.TTY() {
		return red + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightRedf prints and formats msg as red text
func LightRedf(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightRed + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Yellowf prints and formats msg as red text
func Yellowf(msg string, args ...interface{}) string {
	if is.TTY() {
		return yellow + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Grayf prints and formats msg as red text
func Grayf(msg string, args ...interface{}) string {
	if is.TTY() {
		return gray + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Bluef prints and formats msg as red text
func Bluef(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightBlue + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Greenf prints and formats msg as green text
func Greenf(msg string, args ...interface{}) string {
	if is.TTY() {
		return green + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// Magentaf prints and formats msg as green text
func Magentaf(msg string, args ...interface{}) string {
	if is.TTY() {
		return magenta + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightGreenf prints and formats msg as light green text
func LightGreenf(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightGreen + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightCyanf prints and formats msg as light cyan text
func LightCyanf(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightCyan + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}
