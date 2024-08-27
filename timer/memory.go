package timer

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/flanksource/commons/duration"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/text"
	"github.com/shirou/gopsutil/v3/process"
)

type MemoryTimer struct {
	startTime time.Time
	start     *runtime.MemStats
	process   *process.Process
	peakRss   int64
}

func age(d time.Duration) string {
	if d.Milliseconds() == 0 {
		return "0ms"
	}
	if d.Milliseconds() < 1000 {
		return fmt.Sprintf("%0.dms", d.Milliseconds())
	}

	return duration.Duration(d).String()
}

func NewMemoryTimer() MemoryTimer {
	m := MemoryTimer{startTime: time.Now()}
	if logger.IsTraceEnabled() {
		s := runtime.MemStats{}
		runtime.ReadMemStats(&s)
		m.start = &s
		m.process, _ = process.NewProcess(int32(os.Getpid()))
	}
	return m
}

func (m *MemoryTimer) End() string {
	d := age(time.Since(m.startTime))
	if m.start == nil {
		return d
	}

	var rss int64
	if m.process != nil {
		mem, _ := m.process.MemoryInfo()
		rss = int64(mem.RSS)
		if rss > m.peakRss {
			m.peakRss = rss
		}
	}

	end := runtime.MemStats{}
	runtime.ReadMemStats(&end)
	allocs := end.Mallocs - m.start.Mallocs
	heap := end.HeapAlloc - m.start.HeapAlloc
	totalheap := end.TotalAlloc - m.start.TotalAlloc
	gc := end.NumGC - m.start.NumGC

	return fmt.Sprintf(
		"%s (allocs=%s, heap_allocs=%s heap_increase=%s, gc_count=%s rss=%s peak=%s)",
		d,
		text.HumanizeInt(allocs),
		text.HumanizeBytes(totalheap),
		text.HumanizeBytes(heap),
		text.HumanizeInt(gc),
		text.HumanizeBytes(rss),
		text.HumanizeBytes(m.peakRss),
	)
}
