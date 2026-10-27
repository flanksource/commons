// Package yamlcolor opts commons logging into YAML formatting and highlighting.
package yamlcolor

import (
	"encoding/json"
	"github.com/flanksource/commons/logger"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/printer"
	"sigs.k8s.io/yaml"
)

func init() {
	logger.RegisterYAMLFormatter(logger.YAMLFormatter{
		Marshal: func(value any) ([]byte, error) {
			data, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			return yaml.JSONToYAML(data)
		},
		Highlight: highlight,
	})
}

func highlight(value string) string {
	var p printer.Printer
	p.Bool = func() *printer.Property { return &printer.Property{Prefix: "\x1b[95m", Suffix: "\x1b[0m"} }
	p.Number = p.Bool
	p.MapKey = func() *printer.Property { return &printer.Property{Prefix: "\x1b[96m", Suffix: "\x1b[0m"} }
	p.Anchor = func() *printer.Property { return &printer.Property{Prefix: "\x1b[93m", Suffix: "\x1b[0m"} }
	p.Alias = p.Anchor
	p.String = func() *printer.Property { return &printer.Property{Prefix: "\x1b[92m", Suffix: "\x1b[0m"} }
	return p.PrintTokens(lexer.Tokenize(value))
}
