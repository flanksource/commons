package logger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kr/text"
	"sigs.k8s.io/yaml"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/printer"
)

const escape = "\x1b"

func format(attr color.Attribute) string {
	return fmt.Sprintf("%s[%dm", escape, attr)
}

func Pretty(v any) string {
	if m, ok := v.(map[string]any); ok {
		v = StripSecretsFromMap(m)
	}

	b, _ := json.MarshalIndent(v, "  ", "  ")
	b, _ = yaml.JSONToYAML(b)
	return text.Indent(PrettyYAML(strings.TrimSpace(string(b))), "  ")

}

func PrettyYAML(s string) string {
	if !isTTY || !flags.color || IsJsonLogs() {
		return s
	}
	tokens := lexer.Tokenize(s)
	var p printer.Printer
	p.LineNumber = false
	p.Bool = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiMagenta),
			Suffix: format(color.Reset),
		}
	}
	p.Number = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiMagenta),
			Suffix: format(color.Reset),
		}
	}
	p.MapKey = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiCyan),
			Suffix: format(color.Reset),
		}
	}
	p.Anchor = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.Alias = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.String = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiGreen),
			Suffix: format(color.Reset),
		}
	}
	return p.PrintTokens(tokens)
}
