package is

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "terminal detection")
}

var _ = Describe("TTY", func() {
	It("reports no terminal when stdout cannot be inspected", func() {
		file, err := os.CreateTemp(GinkgoT().TempDir(), "stdout")
		Expect(err).NotTo(HaveOccurred())
		Expect(file.Close()).To(Succeed())
		original := os.Stdout
		os.Stdout = file
		defer func() { os.Stdout = original }()
		Expect(TTY()).To(BeFalse())
	})
})
