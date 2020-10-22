package console

import (
	"fmt"
	"io"
)

// TestResults contains the status of testing
type TestResults struct {
	PassCount int
	FailCount int
	SkipCount int
	Tests     []JUnitTestCase
	Name      string
	Writer    io.Writer
	Retries   int
}

func (c *TestResults) Append(other *TestResults) {
	c.Tests = append(c.Tests, other.Tests...)
	c.FailCount += other.FailCount
	c.PassCount += other.PassCount
	c.SkipCount += other.SkipCount
}

func (c *TestResults) Retry() {
	c.PassCount = 0
	c.SkipCount = 0
	c.FailCount = 0
	c.Retries++
	c.Tests = make([]JUnitTestCase, 0)
}

func (c TestResults) ToXML() (string, error) {
	var tests []JUnitTestCase

	for _, test := range c.Tests {
		if test.Classname == "" {
			test.Classname = c.Name
		} else {
			test.Classname = c.Name + "." + test.Classname
		}
		tests = append(tests, test)
	}
	return JUnitTestSuites{
		Suites: []JUnitTestSuite{
			JUnitTestSuite{
				Name:      c.Name,
				TestCases: tests,
				Tests:     c.PassCount + c.FailCount + c.SkipCount,
				Failures:  c.FailCount,
			},
		},
	}.ToXML()
}

func (c TestResults) String() string {
	s := ""
	if c.PassCount > 0 {
		s += Greenf("%d passed", c.PassCount)
	}
	if c.SkipCount > 0 {
		if s != "" {
			s += " "
		}
		s += Magentaf("%d skipped", c.SkipCount)
	}
	if c.FailCount > 0 {
		if s != "" {
			s += " "
		}
		s += Redf("%d failed", c.FailCount)
	}
	if c.Retries > 0 {
		if s != "" {
			s += " "
		}
		s += Redf("after %d retries", c.Retries)
	}
	return s
}

// Done prints the test results to stdout
func (c *TestResults) Done() {
	c.Println(c.String())
}

func NewTestResults(name string, writer io.Writer) TestResults {
	return TestResults{
		Name:   name,
		Writer: writer,
	}
}

func (c *TestResults) Println(s string) {
	fmt.Fprintln(c.Writer, s)
}

var levels = map[string]string{
	"INFO":  LightCyanf("[INFO]  "),
	"DEBUG": DarkF("[DEBUG] "),
	"TRACE": Grayf("[TRACE] "),
	"ERROR": Redf("[ERROR] "),
	"FATAL": Redf("[FATAL] "),
	"WARN":  Yellowf("[WARN]  "),
}

func (c *TestResults) log(level string, s string, args ...interface{}) {
	fmt.Fprintf(c.Writer, levels[level]+s+"\n", args...)
}
func (c *TestResults) Printf(s string, args ...interface{}) {
	c.log("INFO", s, args...)
}
func (c *TestResults) Infof(s string, args ...interface{}) {
	c.log("INFO", s, args...)
}
func (c *TestResults) Debugf(s string, args ...interface{}) {
	c.log("DEBUG", s, args...)
}
func (c *TestResults) Errorf(s string, args ...interface{}) {
	c.log("ERROR", s, args...)
}
func (c *TestResults) Warnf(s string, args ...interface{}) {
	c.log("WARN", s, args...)
}
func (c *TestResults) Tracef(s string, args ...interface{}) {
	c.log("TRACE", s, args...)
}

// Passf reports a new passing test
func (c *TestResults) Passf(name, msg string, args ...interface{}) {
	c.Tests = append(c.Tests, JUnitTestCase{
		Classname: name,
		Name:      fmt.Sprintf(msg, args...),
	})
	c.PassCount++
	c.Println(Greenf(" [pass] "+msg, args...))
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
	c.Println(Redf(" [fail] "+msg, args...))
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
	c.Println(LightCyanf(" [skip] "+msg, args...))
}

func (c *TestResults) SuiteName(name string) *TestResults {
	c.Name = name
	return c
}
