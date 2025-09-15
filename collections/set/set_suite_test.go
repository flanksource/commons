package set

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Set Suite")
}