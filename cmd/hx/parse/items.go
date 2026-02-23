package parse

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ParsedItems struct {
	Headers     map[string]string
	QueryParams map[string]string
	Body        map[string]any
}

func Items(args []string) (*ParsedItems, error) {
	result := &ParsedItems{
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
		Body:        make(map[string]any),
	}

	for _, arg := range args {
		if err := parseItem(arg, result); err != nil {
			return nil, fmt.Errorf("invalid item %q: %w", arg, err)
		}
	}

	return result, nil
}

func parseItem(arg string, result *ParsedItems) error {
	// Order matters: check `:=` and `==` before `=` and `:`

	if idx := strings.Index(arg, ":="); idx > 0 {
		key := arg[:idx]
		val := arg[idx+2:]
		var parsed any
		if err := json.Unmarshal([]byte(val), &parsed); err != nil {
			return fmt.Errorf("invalid JSON for key %q: %w", key, err)
		}
		result.Body[key] = parsed
		return nil
	}

	if idx := strings.Index(arg, "=="); idx > 0 {
		result.QueryParams[arg[:idx]] = arg[idx+2:]
		return nil
	}

	if idx := strings.Index(arg, "="); idx > 0 {
		result.Body[arg[:idx]] = arg[idx+1:]
		return nil
	}

	// Header: `Key:Value` — key must not contain `=` and must have a colon
	// Distinguish from URLs by requiring no `//` after colon
	if idx := strings.Index(arg, ":"); idx > 0 {
		rest := arg[idx+1:]
		if !strings.HasPrefix(rest, "//") {
			result.Headers[arg[:idx]] = strings.TrimSpace(rest)
			return nil
		}
	}

	return fmt.Errorf("cannot parse data item (expected key=value, key:=json, key==param, or Header:Value)")
}

func (p *ParsedItems) HasBody() bool {
	return len(p.Body) > 0
}

func (p *ParsedItems) BodyJSON() ([]byte, error) {
	if len(p.Body) == 0 {
		return nil, nil
	}
	return json.Marshal(p.Body)
}
