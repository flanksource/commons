package properties

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("properties file polling", func() {
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
