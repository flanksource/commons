//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package timer

import (
	"runtime"
	"syscall"
)

// Getrusage reports the process lifetime peak; Darwin reports bytes, BSD/Linux KiB.
func peakResidentMemory() (int64, error) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0, err
	}
	bytes := int64(usage.Maxrss)
	if runtime.GOOS != "darwin" {
		bytes *= 1024
	}
	return bytes, nil
}
