package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
)

// YAMLFormatter installs optional YAML serialization and syntax highlighting.
type YAMLFormatter struct {
	Marshal   func(any) ([]byte, error)
	Highlight func(string) string
}

var yamlFormatter atomic.Pointer[YAMLFormatter]

func RegisterYAMLFormatter(formatter YAMLFormatter) {
	if formatter.Marshal == nil || formatter.Highlight == nil {
		panic("YAML formatter requires Marshal and Highlight")
	}
	yamlFormatter.Store(&formatter)
}

func Pretty(v any) string {
	if m, ok := v.(map[string]any); ok {
		v = StripSecretsFromMap(m)
	}
	var data []byte
	var err error
	if formatter := yamlFormatter.Load(); formatter != nil {
		data, err = formatter.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		return fmt.Sprintf("<format error: %v> %#v", err, v)
	}
	return indent(PrettyYAML(strings.TrimSpace(string(data))), "  ")
}

func PrettyYAML(s string) string {
	if isTTY && flags.color && !IsJsonLogs() {
		if formatter := yamlFormatter.Load(); formatter != nil {
			return formatter.Highlight(s)
		}
	}
	return s
}

func indent(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
