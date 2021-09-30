package text

import (
	"fmt"
	"time"

	"github.com/antonmedv/expr"
)

func MakeExpressionEnvs(envs map[string]interface{}) map[string]interface{} {
	for name, funcMap := range GetTemplateFuncs() {
		envs[name] = funcMap
	}
	envs["Sprintf"] = fmt.Sprintf
	envs["Now"] = time.Now
	envs["Date"] = Date
	envs["Duration"] = Duration
	envs["Equal"] = EqualTime
	envs["Before"] = Before
	envs["BeforeOrEqual"] = BeforeOrEqual
	envs["After"] = After
	envs["AfterOrEqual"] = AfterOrEqual
	envs["Add"] = Add
	envs["Sub"] = Sub
	envs["EqualDuration"] = EqualDuration
	envs["BeforeDuration"] = BeforeDuration
	envs["BeforeOrEqualDuration"] = BeforeOrEqualDuration
	envs["AfterDuration"] = AfterDuration
	envs["AfterOrEqualDuration"] = AfterOrEqualDuration
	envs["Age"] = Age
	return envs
}

func MakeExpressionOptions(envs map[string]interface{}) []expr.Option {
	// Operators override for date comprising.
	envs = MakeExpressionEnvs(envs)
	options := []expr.Option{
		expr.Env(envs),
		// Operators override for date comprising.
		expr.Operator("==", "Equal"),
		expr.Operator("<", "Before"),
		expr.Operator("<=", "BeforeOrEqual"),
		expr.Operator(">", "After"),
		expr.Operator(">=", "AfterOrEqual"),

		// Time and duration manipulation.
		expr.Operator("+", "Add"),
		expr.Operator("-", "Sub"),

		// Operators override for duration comprising.
		expr.Operator("==", "EqualDuration"),
		expr.Operator("<", "BeforeDuration"),
		expr.Operator("<=", "BeforeOrEqualDuration"),
		expr.Operator(">", "AfterDuration"),
		expr.Operator(">=", "AfterOrEqualDuration"),
	}
	return options
}

func Date(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
func Duration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func EqualTime(a, b time.Time) bool                 { return a.Equal(b) }
func Before(a, b time.Time) bool                    { return a.Before(b) }
func BeforeOrEqual(a, b time.Time) bool             { return a.Before(b) || a.Equal(b) }
func After(a, b time.Time) bool                     { return a.After(b) }
func AfterOrEqual(a, b time.Time) bool              { return a.After(b) || a.Equal(b) }
func Add(a time.Time, b time.Duration) time.Time    { return a.Add(b) }
func Sub(a, b time.Time) time.Duration              { return a.Sub(b) }
func EqualDuration(a, b time.Duration) bool         { return a == b }
func BeforeDuration(a, b time.Duration) bool        { return a < b }
func BeforeOrEqualDuration(a, b time.Duration) bool { return a <= b }
func AfterDuration(a, b time.Duration) bool         { return a > b }
func AfterOrEqualDuration(a, b time.Duration) bool  { return a >= b }
func Age(s string) time.Duration {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return time.Since(t)
}
