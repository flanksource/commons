package console

import (
	"fmt"
)

type TestResults struct {
	PassCount int
	FailCount int
	SkipCount int
}

func (c TestResults) String() string {
	return fmt.Sprintf("  %d passed, %d skipped, %d failed\n", c.PassCount, c.SkipCount, c.FailCount)
}

func (c *TestResults) Done() {
	fmt.Println(c.String())
}

func (c *TestResults) Passf(name, msg string, args ...interface{}) {
	c.PassCount++
	fmt.Println(Greenf(" [pass] "+msg, args...))

}
func (c *TestResults) Failf(name, msg string, args ...interface{}) {
	c.FailCount++
	fmt.Println(Redf(" [fail] "+msg, args...))
}
func (c *TestResults) Skipf(name, msg string, args ...interface{}) {
	c.SkipCount++
	fmt.Println(LightCyanf(" [skip] "+msg, args...))
}
