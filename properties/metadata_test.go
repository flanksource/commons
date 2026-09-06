package properties

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("property metadata", func() {
	var store *Properties

	BeforeEach(func() {
		store = &Properties{m: make(map[string]string)}
		previous := commandlineProperties
		commandlineProperties = nil
		DeferCleanup(func() { commandlineProperties = previous })
	})

	It("lists typed defaults even when no value was explicitly stored", func() {
		Expect(store.String("localhost", "server.host")).To(Equal("localhost"))
		Expect(store.Int(8080, "server.port")).To(Equal(8080))
		Expect(store.Bytes(4*1024*1024, "server.max-body-size")).To(Equal(4 * 1024 * 1024))
		Expect(store.On(true, "server.enabled")).To(BeTrue())
		Expect(store.Duration(30*time.Second, "server.timeout")).To(Equal(30 * time.Second))
		Expect(store.LogLevel("info", []string{"error", "info", "debug"}, "log.level")).To(Equal("info"))
		intervals, err := store.TimeIntervals("business_hours")
		Expect(err).NotTo(HaveOccurred())
		Expect(intervals).To(BeEmpty())

		Expect(store.List()).To(Equal([]Property{
			{Key: "log.level", Value: "info", Default: "info", HasDefault: true, Type: PropertyTypeLogLevel, Options: []string{"error", "info", "debug"}, Source: PropertySourceDefault},
			{Key: "server.enabled", Value: "true", Default: "true", HasDefault: true, Type: PropertyTypeBool, Source: PropertySourceDefault},
			{Key: "server.host", Value: "localhost", Default: "localhost", HasDefault: true, Type: PropertyTypeString, Source: PropertySourceDefault},
			{Key: "server.max-body-size", Value: "4194304", Default: "4194304", HasDefault: true, Type: PropertyTypeBytes, Source: PropertySourceDefault},
			{Key: "server.port", Value: "8080", Default: "8080", HasDefault: true, Type: PropertyTypeInt, Source: PropertySourceDefault},
			{Key: "server.timeout", Value: "30s", Default: "30s", HasDefault: true, Type: PropertyTypeDuration, Source: PropertySourceDefault},
			{Key: "time_interval.business_hours", Type: PropertyTypeTimeIntervals, Source: PropertySourceUnset},
		}))
	})

	It("reports explicit and command-line values with their declared metadata", func() {
		store.Set("server.port", "9090")
		Expect(store.Int(8080, "server.port")).To(Equal(9090))
		commandlineProperties = map[string]string{"server.host": "example.internal"}
		Expect(store.String("localhost", "server.host")).To(Equal("example.internal"))

		Expect(store.List()).To(Equal([]Property{
			{Key: "server.host", Value: "example.internal", Default: "localhost", HasDefault: true, Type: PropertyTypeString, Source: PropertySourceCommandLine, ReadOnly: true},
			{Key: "server.port", Value: "9090", Default: "8080", HasDefault: true, Type: PropertyTypeInt, Source: PropertySourceRuntime},
		}))
	})

	It("includes untyped values as runtime strings", func() {
		store.Set("custom.property", "value")

		Expect(store.List()).To(Equal([]Property{{
			Key: "custom.property", Value: "value", Type: PropertyTypeString, Source: PropertySourceRuntime,
		}}))
	})
})
