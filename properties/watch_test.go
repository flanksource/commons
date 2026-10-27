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
})
