package logger_test

import (
	"github.com/flanksource/commons/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("logger flag API", func() {
	It("retains pflag registration and original typed callbacks", func() {
		previous := logger.StandardLogger().GetLevel()
		DeferCleanup(func() { logger.StandardLogger().SetLogLevel(previous) })
		flags := pflag.NewFlagSet("logger", pflag.ContinueOnError)
		func(bind func(*pflag.FlagSet)) { bind(flags) }(logger.BindFlags)
		Expect(flags.Parse([]string{"-vv", "--color=false"})).To(Succeed())
		level, err := flags.GetCount("loglevel")
		Expect(err).NotTo(HaveOccurred())
		Expect(level).To(Equal(2))
		color, err := flags.GetBool("color")
		Expect(err).NotTo(HaveOccurred())
		Expect(color).To(BeFalse())
		func(use func(*pflag.FlagSet)) { use(flags) }(logger.UseCobraFlags)
	})
})
