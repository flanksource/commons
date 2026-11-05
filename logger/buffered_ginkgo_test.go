package logger

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type bufferedLine struct {
	Seq     uint64
	Message string
}

func bufferedLines(entries []BufferedLogEntry) []bufferedLine {
	lines := make([]bufferedLine, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, bufferedLine{Seq: entry.Seq, Message: entry.Message})
	}
	return lines
}

var _ = ginkgo.Describe("BufferedLogger sequence", func() {
	ginkgo.It("numbers entries in insertion order across levels", func() {
		buffered := NewBufferedLogger(10)
		buffered.Infof("first")
		buffered.Warnf("second")
		buffered.Errorf("third")
		buffered.Infof("fourth")

		gomega.Expect(bufferedLines(buffered.GetLogs())).To(gomega.Equal([]bufferedLine{
			{Seq: 1, Message: "first"},
			{Seq: 2, Message: "second"},
			{Seq: 3, Message: "third"},
			{Seq: 4, Message: "fourth"},
		}))
	})

	ginkgo.It("keeps sequence order when one level evicts its oldest entries", func() {
		const infoRetained = 2
		buffered := NewBufferedLoggerWithRetention(RetentionConfig{Info: infoRetained, Warn: 5})
		buffered.Infof("info-1")
		buffered.Warnf("warn-1")
		buffered.Infof("info-2")
		buffered.Warnf("warn-2")
		buffered.Infof("info-3")

		gomega.Expect(bufferedLines(buffered.GetLogs())).To(gomega.Equal([]bufferedLine{
			{Seq: 2, Message: "warn-1"},
			{Seq: 3, Message: "info-2"},
			{Seq: 4, Message: "warn-2"},
			{Seq: 5, Message: "info-3"},
		}))
	})

	ginkgo.It("records the last cleared sequence and keeps numbering after a clear", func() {
		buffered := NewBufferedLogger(10)
		gomega.Expect(buffered.ClearedThrough()).To(gomega.BeZero())
		buffered.Infof("one")
		buffered.Infof("two")
		buffered.Infof("three")

		buffered.ClearLogs()
		gomega.Expect(buffered.ClearedThrough()).To(gomega.Equal(uint64(3)))
		gomega.Expect(buffered.GetLogs()).To(gomega.BeEmpty())

		buffered.Infof("four")
		gomega.Expect(bufferedLines(buffered.GetLogs())).To(gomega.Equal([]bufferedLine{{Seq: 4, Message: "four"}}))
	})
})
