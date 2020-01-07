package console

import (
	"fmt"
)

// TestResults contains the status of testing
type TestResults struct {
	PassCount int
	FailCount int
	SkipCount int
	Tests     []JUnitTestCase
	Name string
}

func (c TestResults) ToXML() (string, error) {
	return JUnitTestSuites{
		Suites: []JUnitTestSuite{
			JUnitTestSuite{
				Name: c.Name,
				TestCases: c.Tests,
				Tests:     c.PassCount + c.FailCount + c.SkipCount,
				Failures:  c.FailCount,
			},
		},
	}.ToXML()

}
func (c TestResults) String() string {
	return fmt.Sprintf("  %d passed, %d skipped, %d failed\n", c.PassCount, c.SkipCount, c.FailCount)
}

// Done prints the test results to stdout
func (c *TestResults) Done() {
	fmt.Println(c.String())
}

// Passf reports a new passing test
func (c *TestResults) Passf(name, msg string, args ...interface{}) {
	c.Tests = append(c.Tests, JUnitTestCase{
		Classname: name,
		Name:      fmt.Sprintf(msg, args...),
	})
	c.PassCount++
	fmt.Println(Greenf(" [pass] "+msg, args...))
}

// Failf reports a new failing test
func (c *TestResults) Failf(name, msg string, args ...interface{}) {
	c.Tests = append(c.Tests, JUnitTestCase{
		Classname: name,
		Name:      fmt.Sprintf(msg, args...),
		Failure: &JUnitFailure{
			Message: fmt.Sprintf(msg, args...),
		},
	})
	c.FailCount++
	fmt.Println(Redf(" [fail] "+msg, args...))
}

// Skipf reports a new skipped test
func (c *TestResults) Skipf(name, msg string, args ...interface{}) {
	c.Tests = append(c.Tests, JUnitTestCase{
		Classname: name,
		Name:      fmt.Sprintf(msg, args...),
		SkipMessage: &JUnitSkipMessage{
			Message: fmt.Sprintf(msg, args...),
		},
	})
	c.SkipCount++
	fmt.Println(LightCyanf(" [skip] "+msg, args...))
}

func (c *TestResults) SuiteName(name string) *TestResults {
	c.Name = name
	return c
}
