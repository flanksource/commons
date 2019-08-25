package console

import (
	"fmt"
	"github.com/moshloop/commons/is"
)

var (
	reset        = "\x1b[0m"
	red          = "\x1b[31m"
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
)

func Redf(msg string, args ...interface{}) string {
	if is.TTY() {
		return red + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

func Greenf(msg string, args ...interface{}) string {
	if is.TTY() {
		return green + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

func LightGreenf(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightGreen + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}

func LightCyanf(msg string, args ...interface{}) string {
	if is.TTY() {
		return lightCyan + fmt.Sprintf(msg, args...) + reset
	}
	return fmt.Sprintf(msg, args...)
}
