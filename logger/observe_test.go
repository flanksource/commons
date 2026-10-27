package logger

import (
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("logger observer", func() {
	It("observes a named message and stops after release", func() {
		var records []ObservedLog
		release := Observe(func(record ObservedLog) { records = append(records, record) })
		log := GetLogger("orders")
		log.Infof("order %d", 7)
		release()
		log.Infof("after release")
		Expect(records).To(HaveLen(1))
		Expect(records[0].Name).To(Equal("orders"))
		Expect(records[0].Message).To(Equal("order 7"))
		Expect(records[0].Level).To(Equal(slog.LevelInfo))
	})
	It("announces loggers created after a listener is registered", func() {
		var names []string
		release := ObserveCreated(func(name string) { names = append(names, name) })
		defer release()
		GetLogger("fresh-example")
		Expect(names).To(ContainElement("fresh-example"))
	})
})
