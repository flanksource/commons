package text

import (
	"fmt"
	"testing"

	"github.com/antonmedv/expr"
)

var result = map[string]interface{}{
	"result": map[string]string{"started_at": "2021-09-21T10:02:05Z"},
}

func TestExpressionsTimeFunctions(t *testing.T) {
	var expression = `Age(result["started_at"]) > Duration("24h")`
	program, err := expr.Compile(expression, GetTestExpresionOptions(result)...)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := expr.Run(program, GetTestExpressionEnvs(result))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(output)
	if output != true {
		t.Error("Expression should be true")
		return
	}

}
