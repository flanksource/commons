package timer

import (
	"fmt"
	"time"
)

type Timer struct {
	Start, End time.Time
}

func (t Timer) Elapsed() float64 {
	since := t.End
	if since.IsZero() {
		since = time.Now()
	}
	return float64(since.Sub(t.Start).Milliseconds())
}

func (t Timer) Millis() int64 {
	since := t.End
	if since.IsZero() {
		since = time.Now()
	}
	return time.Since(t.Start).Milliseconds()
}

func (t Timer) Stop() {
	t.End = time.Now()
}

func (t Timer) String() string {
	millis := t.Millis()
	if millis > 60*1000 {
		return fmt.Sprintf("%dm", millis/60/1000)
	} else if millis > 1000 {
		return fmt.Sprintf("%ds", millis/1000)
	}
	return fmt.Sprintf("%dms", millis)
}

func NewTimer() Timer {
	return Timer{Start: time.Now()}
}
