package har_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHARSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HAR Suite")
}
