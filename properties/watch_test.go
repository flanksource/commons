package properties

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("properties file polling", func() {
	It("keeps command-line overrides when an optional file appears", func() {
		isolateCommandline()
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		flags := pflag.NewFlagSet("watch", pflag.ContinueOnError)
		BindFlags(flags)
		Expect(flags.Parse([]string{"-P", "server.port=7070"})).To(Succeed())
		filename := filepath.Join(GinkgoT().TempDir(), "optional.properties")
		store := &Properties{m: make(map[string]string)}
		Expect(store.LoadFile(filename)).To(Succeed())
		DeferCleanup(store.Watch())
		Expect(os.WriteFile(filename, []byte("server.port=9090\nloaded=true\n"), 0600)).To(Succeed())
		Eventually(func() string { return store.Get("loaded") }, time.Second).Should(Equal("true"))
		Expect(store.Get("server.port")).To(Equal("7070"))
	})
	It("loads an initially absent file and preserves values through deletion and recreation", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		filename := filepath.Join(GinkgoT().TempDir(), "optional.properties")
		store := &Properties{m: make(map[string]string)}
		Expect(store.LoadFile(filename)).To(Succeed())
		DeferCleanup(func() {
			if store.close != nil {
				store.close()
			}
		})
		Expect(store.filename).To(Equal(filename))
		Expect(os.WriteFile(filename, []byte("server.port=8080\n"), 0600)).To(Succeed())
		Eventually(func() string { return store.Get("server.port") }, time.Second).Should(Equal("8080"))
		Expect(os.Remove(filename)).To(Succeed())
		Consistently(func() string { return store.Get("server.port") }, 80*time.Millisecond).Should(Equal("8080"))
		Expect(os.WriteFile(filename, []byte("server.port=9090\n"), 0600)).To(Succeed())
		Eventually(func() string { return store.Get("server.port") }, time.Second).Should(Equal("9090"))
		store.close()
		store.close()
	})
	It("loads a file created between the failed open and the start of watching", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		filename := filepath.Join(GinkgoT().TempDir(), "late.properties")
		Expect(os.WriteFile(filename, []byte("server.port=9090\n"), 0600)).To(Succeed())
		store := &Properties{m: make(map[string]string)}
		store.watchFile(filename, nil)
		DeferCleanup(func() { store.close() })
		Eventually(func() string { return store.Get("server.port") }, time.Second).Should(Equal("9090"))
	})
	It("starts watching when the configured file has disappeared", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		filename := filepath.Join(GinkgoT().TempDir(), "removed.properties")
		store := &Properties{m: make(map[string]string), filename: filename}
		stop := store.Watch()
		DeferCleanup(stop)
		Expect(os.WriteFile(filename, []byte("server.port=9090\n"), 0600)).To(Succeed())
		Eventually(func() string { return store.Get("server.port") }, time.Second).Should(Equal("9090"))
	})
	It("reloads a file after atomic replacement and stops idempotently", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		filename := filepath.Join(GinkgoT().TempDir(), "app.properties")
		Expect(os.WriteFile(filename, []byte("server.port=8080\n"), 0600)).To(Succeed())
		store := &Properties{m: make(map[string]string)}
		Expect(store.LoadFile(filename)).To(Succeed())
		closeWatch := store.Watch()
		DeferCleanup(closeWatch)
		Expect(store.Get("server.port")).To(Equal("8080"))
		replacement := filename + ".new"
		Expect(os.WriteFile(replacement, []byte("server.port=9090\n"), 0600)).To(Succeed())
		Expect(os.Rename(replacement, filename)).To(Succeed())
		Eventually(func() string { return store.Get("server.port") }, time.Second).Should(Equal("9090"))
		closeWatch()
		closeWatch()
	})

	It("defaults to a five-second polling interval", func() {
		Expect(WatchInterval).To(Equal(5 * time.Second))
	})

	It("does not reload when polling is disabled", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "0")
		filename := filepath.Join(GinkgoT().TempDir(), "app.properties")
		Expect(os.WriteFile(filename, []byte("server.port=8080\n"), 0600)).To(Succeed())
		store := &Properties{m: make(map[string]string)}
		Expect(store.LoadFile(filename)).To(Succeed())
		closeWatch := store.Watch()
		DeferCleanup(closeWatch)
		Expect(os.WriteFile(filename, []byte("server.port=9090\n"), 0600)).To(Succeed())
		Consistently(func() string { return store.Get("server.port") }, 5500*time.Millisecond).Should(Equal("8080"))
		closeWatch()
		closeWatch()
	})

	It("refuses to watch before a file has been loaded", func() {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", "20ms")
		store := &Properties{m: make(map[string]string)}
		Expect(func() { store.Watch() }).To(PanicWith(MatchError("watch properties: no properties file has been loaded")))
	})

	DescribeTable("rejects invalid polling intervals", func(value string) {
		GinkgoT().Setenv("COMMONS_PROPERTIES_WATCH_DURATION", value)
		filename := filepath.Join(GinkgoT().TempDir(), "app.properties")
		Expect(os.WriteFile(filename, []byte("server.port=8080\n"), 0600)).To(Succeed())
		store := &Properties{m: make(map[string]string)}
		Expect(func() { _ = store.LoadFile(filename) }).To(PanicWith(MatchError(ContainSubstring("COMMONS_PROPERTIES_WATCH_DURATION"))))
	},
		Entry("invalid duration", "invalid"),
		Entry("negative duration", "-1s"),
		Entry("empty duration", ""),
	)
})
