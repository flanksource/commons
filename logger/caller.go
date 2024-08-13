package logger

import (
	"runtime"
	"strconv"
	"strings"
)

var SkipFrameSuffixes = []string{
	"logger/slog.go",
	"logger/default.go",
	"logger/caller.go",
	"gorm/logger.go",
	"golang.org/toolchain",
}

var SkipFrameContains = []string{
	"gorm.io",
	"golang.org/toolchain",
}

// Caller return the file name and line number of the current file
func Caller(skip ...int) string {
	pcs := [13]uintptr{}
	start := 0
	if len(skip) > 0 {
		start += skip[0]
	}
	len := runtime.Callers(start, pcs[:])
	frames := runtime.CallersFrames(pcs[:len])
	for i := 0; i < len; i++ {
		// second return value is "more", not "ok"
		frame, _ := frames.Next()
		if !skipFrame(frame) {
			return string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10))
		}
	}
	return ""
}

func Stacktrace() string {
	pcs := [13]uintptr{}
	len := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:len])
	s := ""
	for i := 0; i < len; i++ {
		// second return value is "more", not "ok"
		frame, _ := frames.Next()
		if !skipFrame(frame) {
			s += string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10)) + "\n"
		}
	}
	return s
}

func skipFrame(frame runtime.Frame) bool {
	for _, suffix := range SkipFrameSuffixes {
		if strings.HasSuffix(frame.File, suffix) {
			return true
		}
	}
	for _, val := range SkipFrameContains {
		if strings.Contains(frame.File, val) {
			return true
		}
	}
	return false
}

func CallerPC(skip ...int) uintptr {
	pcs := [13]uintptr{}
	len := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:len])
	for i := 0; i < len; i++ {
		// second return value is "more", not "ok"
		frame, _ := frames.Next()
		if !skipFrame(frame) {
			return pcs[i]
		}
	}
	return pcs[1]
}
