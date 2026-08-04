package help

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHelp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Help Suite")
}
