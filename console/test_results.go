package console

import (
	"fmt"
)

// TestResults contains the status of testing
type TestResults struct {
	PassCount int
	FailCount int
	SkipCount int
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
	c.PassCount++
	fmt.Println(Greenf(" [pass] "+msg, args...))

}

// Failf reports a new failing test
func (c *TestResults) Failf(name, msg string, args ...interface{}) {
	c.FailCount++
	fmt.Println(Redf(" [fail] "+msg, args...))
}

// Skipf reports a new skipped test
func (c *TestResults) Skipf(name, msg string, args ...interface{}) {
	c.SkipCount++
	fmt.Println(LightCyanf(" [skip] "+msg, args...))
}
