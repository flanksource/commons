package properties

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("property removal", func() {
	It("removes the value and notifies listeners", func() {
		store := &Properties{m: make(map[string]string)}
		notifications := 0
		store.listeners = append(store.listeners, func(*Properties) { notifications++ })
		store.Set("example.level", "debug")
		store.Unset("example.level")
		Expect(store.GetAll()).NotTo(HaveKey("example.level"))
		Expect(notifications).To(Equal(2))
	})
})
