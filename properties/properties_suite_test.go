package properties

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProperties(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Properties Suite")
}

// isolateCommandline clears the command-line overrides for one spec and restores them afterwards.
func isolateCommandline() {
	previous := commandline.snapshot()
	commandline.replace(nil)
	DeferCleanup(func() { commandline.replace(previous) })
}
