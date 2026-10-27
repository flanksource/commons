//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package timer

import "fmt"

func peakResidentMemory() (int64, error) {
	return 0, fmt.Errorf("Getrusage is not supported on this platform")
}
