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
		Entry("numeric day", `-1`, &DayOfMonthRange{}, &DayOfMonthRange{InclusiveRange{Begin: -1, End: -1}}),
		Entry("numeric month", `11`, &MonthRange{}, &MonthRange{InclusiveRange{Begin: 11, End: 11}}),
		Entry("numeric year", `2026`, &YearRange{}, &YearRange{InclusiveRange{Begin: 2026, End: 2026}}),
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
		Entry("fractional year", `2026.5`, &YearRange{}),
		Entry("exponent year", `2e3`, &YearRange{}),
		Entry("null year", `null`, &YearRange{}),
		Entry("boolean month", `true`, &MonthRange{}),
		Entry("numeric zero day", `0`, &DayOfMonthRange{}),
		Entry("numeric day out of range", `32`, &DayOfMonthRange{}),
		Entry("numeric month zero", `0`, &MonthRange{}),
		Entry("numeric month thirteen", `13`, &MonthRange{}),
		Entry("quoted month thirteen", `"13"`, &MonthRange{}),
		Entry("month range ending out of range", `"january:13"`, &MonthRange{}),
		Entry("integer overflow", `99999999999999999999999999`, &YearRange{}),
		Entry("array", `[]`, &YearRange{}),
		Entry("object", `{}`, &YearRange{}),
	)
	It("rejects out-of-range months in the interval months list", func() {
		for _, input := range []string{`[{"months":[0]}]`, `[{"months":[13]}]`} {
			var intervals TimeIntervals
			Expect(json.Unmarshal([]byte(input), &intervals)).To(MatchError(ContainSubstring("not a valid month")), input)
		}
	})
	It("decodes complete intervals with numeric and string components identically", func() {
		var numeric, quoted TimeIntervals
		Expect(json.Unmarshal([]byte(`[{"years":[2026],"months":[11],"days_of_month":[2]}]`), &numeric)).To(Succeed())
		Expect(json.Unmarshal([]byte(`[{"years":["2026"],"months":["11"],"days_of_month":["2"]}]`), &quoted)).To(Succeed())
		Expect(numeric).To(Equal(quoted))
	})
})
