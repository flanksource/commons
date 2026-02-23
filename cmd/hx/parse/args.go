package parse

import (
	"fmt"
	"strings"
)

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true, "TRACE": true,
}

type Args struct {
	Method string
	URL    string
	Items  []string
}

func PositionalArgs(args []string) (*Args, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("URL is required")
	}

	result := &Args{}

	if httpMethods[strings.ToUpper(args[0])] {
		result.Method = strings.ToUpper(args[0])
		args = args[1:]
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("URL is required")
	}

	result.URL = args[0]
	result.Items = args[1:]

	return result, nil
}

func (a *Args) EffectiveMethod(hasBody bool, methodOverride string) string {
	if methodOverride != "" {
		return strings.ToUpper(methodOverride)
	}
	if a.Method != "" {
		return a.Method
	}
	if hasBody {
		return "POST"
	}
	return "GET"
}
