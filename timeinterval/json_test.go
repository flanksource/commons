package timeinterval

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJSON(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Time Interval JSON Suite")
}

var _ = Describe("JSON time intervals", func() {
	DescribeTable("decodes and validates interval components", func(input string, target, expected any) {
		Expect(json.Unmarshal([]byte(input), target)).To(Succeed())
		Expect(target).To(Equal(expected))
	},
		Entry("location", `"UTC"`, &Location{}, &Location{time.UTC}),
		Entry("weekdays", `"monday:friday"`, &WeekdayRange{}, &WeekdayRange{InclusiveRange{Begin: 1, End: 5}}),
		Entry("days of month", `"-3:-1"`, &DayOfMonthRange{}, &DayOfMonthRange{InclusiveRange{Begin: -3, End: -1}}),
		Entry("months", `"january:march"`, &MonthRange{}, &MonthRange{InclusiveRange{Begin: 1, End: 3}}),
		Entry("years", `"2024:2026"`, &YearRange{}, &YearRange{InclusiveRange{Begin: 2024, End: 2026}}),
		Entry("times", `{"start_time":"09:00","end_time":"17:00"}`, &TimeRange{}, &TimeRange{StartMinute: 540, EndMinute: 1020}),
	)

	DescribeTable("rejects invalid interval components", func(input string, target any) {
		Expect(json.Unmarshal([]byte(input), target)).To(HaveOccurred())
	},
		Entry("unknown location", `"Example/Invalid"`, &Location{}),
		Entry("reversed weekdays", `"friday:monday"`, &WeekdayRange{}),
		Entry("zero day of month", `"0"`, &DayOfMonthRange{}),
		Entry("unknown month", `"unknown"`, &MonthRange{}),
		Entry("reversed years", `"2026:2024"`, &YearRange{}),
		Entry("reversed times", `{"start_time":"17:00","end_time":"09:00"}`, &TimeRange{}),
		Entry("YAML scalar", `monday`, &WeekdayRange{}),
	)
})
