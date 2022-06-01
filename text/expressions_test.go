package text

import (
	"fmt"
	"testing"

	"github.com/antonmedv/expr"
)

type Expression string
type Output interface{}

type Fixtures map[Expression]Output

var fixtures = Fixtures{
	`Age(result["started_at"]) > Duration("24h")`: true,
	`humanizeBytes(result["size_string"])`:        "1K",
	`humanizeBytes(result["size_int"])`:           "1K",
	`humanizeTime(Date(result["started_at"]))`:    "2 years ago",
}

var Results = map[string]interface{}{
	"result": map[string]interface{}{
		"started_at":  "2020-09-21T10:02:05Z",
		"size_string": "1024",
		"size_int":    1024,
	},
}

func (fixtures Fixtures) Evaluate() error {
	for expression, output := range fixtures {
		program, err := expr.Compile(string(expression), MakeExpressionOptions(Results)...)
		if err != nil {
			return err
		}
		result, err := expr.Run(program, MakeExpressionEnvs(Results))
		if err != nil {
			return err
		}
		if result != output {
			return fmt.Errorf("expected: %v. Got: %v", output, result)
		}
	}
	return nil
}

func TestExpressions(t *testing.T) {
	for expression, output := range fixtures {
		var fixture Fixtures = map[Expression]Output{}
		fixture[expression] = output
		err := fixture.Evaluate()
		if err != nil {
			t.Errorf("error evaluating expression %q: %v", expression, err)
		}
	}
}
