package text

import (
	"testing"

	"github.com/antonmedv/expr"
)

var results = map[string]interface{}{
	"result": map[string]interface{}{
		"started_at":  "2020-09-21T10:02:05Z",
		"size_string": "1024",
		"size_int":    1024,
	},
}

func TestExpressionsTimeFunctions(t *testing.T) {
	var expression = `Age(result["started_at"]) > Duration("24h")`
	program, err := expr.Compile(expression, GetTestExpresionOptions(results)...)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := expr.Run(program, GetTestExpressionEnvs(results))
	if err != nil {
		t.Error(err)
		return
	}
	if output != true {
		t.Error("Expression should be true")
		return
	}

}

func TestExpressionHumanizeBytesString(t *testing.T) {
	var expression = `humanizeBytes(uint64FromString(result["size_string"]))`
	program, err := expr.Compile(expression, GetTestExpresionOptions(results)...)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := expr.Run(program, GetTestExpressionEnvs(results))
	if err != nil {
		t.Error(err)
		return
	}
	if output != "1K" {
		t.Errorf("Expected 1K, Got: %v", output)
		return
	}
}

func TestExpressionHumanizeBytesInt(t *testing.T) {
	var expression = `humanizeBytes(uint64FromInt(result["size_int"]))`
	program, err := expr.Compile(expression, GetTestExpresionOptions(results)...)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := expr.Run(program, GetTestExpressionEnvs(results))
	if err != nil {
		t.Error(err)
		return
	}
	if output != "1K" {
		t.Errorf("Expected: 1K, Got: %v", output)
		return
	}
}

func TestExpressionHumanizeTime(t *testing.T) {
	var expression = `humanizeTime(Date(result["started_at"]))`
	program, err := expr.Compile(expression, GetTestExpresionOptions(results)...)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := expr.Run(program, GetTestExpressionEnvs(results))
	if err != nil {
		t.Error(err)
		return
	}
	if output != "1 year ago" {
		t.Errorf("expected: '1 year ago' but got: %v", output)
	}
}
