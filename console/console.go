package console

import (
	"fmt"

	"github.com/flanksource/commons/is"
	"github.com/flanksource/commons/logger"
)

var (
	Light       = ";1m"
	Bright      = ";40m"
	Normal      = "m"
	Reset       = "\x1b[0m"
	LightYellow = yellow + Light
	red         = "\x1b[31"
	Red         = red + Normal

	yellow     = "\x1b[33"
	Yellow     = yellow + Normal
	gray       = "\x1b[37"
	Gray       = gray + Normal
	LightGray  = gray + Light
	BrightGray = gray + Bright
	LightRed   = red + Light
	green      = "\x1b[32"
	Green      = green + Normal
	LightGreen = green + Light

	BrightYellow = yellow + Bright

	blue          = "\x1b[34"
	Blue          = blue + Normal
	LightBlue     = blue + Light
	magenta       = "\x1b[35"
	Magenta       = magenta + Normal
	LightMagenta  = magenta + Light
	BrightMagenta = magenta + Bright
	cyan          = "\x1b[36"
	Cyan          = cyan + Normal
	LightCyan     = cyan + Light
	BrightCyan    = cyan + Bright
	white         = "\x1b[38"
	White         = white + Light
	BoldOn        = "\x1b[1m"
	BoldOff       = "\x1b[22m"
	DarkWhite     = "\x1b[38;5;244m"
	BrightWhite   = "\x1b[38;5;244m"
)

var (
	isTTY = is.TTY()
)

func ColorOff() {
	isTTY = false
}

// DarkWhitef prints and formats msg as dark white
func DarkWhitef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return DarkWhite + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// DarkWhitef prints and formats msg as dark white
func LightWhitef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return white + Light + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// DarkWhitef prints and formats msg as dark white
func BrightWhitef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return white + Bright + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// DarkWhitef prints and formats msg as dark white
func DarkF(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return DarkWhite + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// DarkWhitef prints and formats msg as dark white
func BrightF(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return DarkWhite + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Redf prints and formats msg as red text
func Redf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Red + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightRedf prints and formats msg as red text
func LightRedf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightRed + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Yellowf prints and formats msg as red text
func Yellowf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Yellow + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// BrightYellowf prints and formats msg as red text
func BrightYellowf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return BrightYellow + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightYellowf prints and formats msg as red text
func LightYellowf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightYellow + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Grayf prints and formats msg as red text
func Grayf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Gray + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// BrightGrayf prints and formats msg as red text
func BrightGrayf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return BrightGray + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightGrayf prints and formats msg as red text
func LightGrayf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightGray + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Bluef prints and formats msg as red text
func Bluef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Blue + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightBluef prints and formats msg as red text
func LightBluef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightBlue + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// BrightBluef prints and formats msg as red text
func BrightBluef(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return blue + Light + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Greenf prints and formats msg as green text
func Greenf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Green + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// BrightGreenf prints and formats msg as light green text
func BrightGreenf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return green + Bright + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightGreenf prints and formats msg as light green text
func LightGreenf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return green + Light + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightCyanf prints and formats msg as light cyan text
func LightCyanf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightCyan + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightCyanf prints and formats msg as light cyan text
func Cyanf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Cyan + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// LightCyanf prints and formats msg as light cyan text
func BrightCyanf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return BrightCyan + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Magentaf prints and formats msg as green text
func Magentaf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return Magenta + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Magentaf prints and formats msg as green text
func LightMagentaf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return LightMagenta + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}

// Magentaf prints and formats msg as green text
func BrightMagentaf(msg string, args ...interface{}) string {
	if isTTY && !logger.IsJsonLogs() {
		return BrightMagenta + fmt.Sprintf(msg, args...) + Reset
	}
	return fmt.Sprintf(msg, args...)
}
