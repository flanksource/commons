package money

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMoney(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Money Suite")
}
