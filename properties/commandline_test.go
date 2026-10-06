package properties

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("command-line properties", func() {
	const key = "server.host"

	BeforeEach(isolateCommandline)

	newFlagSet := func(name string) *pflag.FlagSet {
		flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
		BindFlags(flags)
		return flags
	}

	It("keeps parsed values when another flag set binds afterwards", func() {
		Expect(newFlagSet("first").Parse([]string{"-P", key + "=example.internal"})).To(Succeed())

		newFlagSet("second")

		Expect(Get(key)).To(Equal("example.internal"))
	})

	It("merges repeated -P flags and overrides runtime values", func() {
		store := &Properties{m: map[string]string{key: "runtime", "server.port": "8080"}}

		Expect(newFlagSet("app").Parse([]string{"-P", key + "=example.internal", "--properties", "log.level=debug"})).To(Succeed())

		Expect(store.GetAll()).To(Equal(map[string]string{key: "example.internal", "server.port": "8080", "log.level": "debug"}))
		Expect(store.m).To(Equal(map[string]string{key: "runtime", "server.port": "8080"}), "GetAll must not write command-line values into the store")
	})

	It("binds and parses flag sets concurrently while values are read", func() {
		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				defer GinkgoRecover()
				Expect(newFlagSet(fmt.Sprintf("app-%d", i)).Parse([]string{"-P", fmt.Sprintf("%s=host-%d", key, i)})).To(Succeed())
			})
			wg.Go(func() {
				_ = Get(key)
				_ = Global.GetAll()
				_ = Global.List()
			})
		}
		wg.Wait()

		Expect(Get(key)).To(MatchRegexp(`^host-\d$`))
	})
})
