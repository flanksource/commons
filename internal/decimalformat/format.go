package decimalformat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var numeric = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func Decode(data []byte) (string, error) {
	if !json.Valid(data) {
		return "", fmt.Errorf("invalid decimal JSON %q", data)
	}
	text := strings.TrimSpace(string(data))
	if strings.HasPrefix(text, "\"") {
		if err := json.Unmarshal(data, &text); err != nil {
			return "", err
		}
	}
	if !numeric.MatchString(text) {
		return "", fmt.Errorf("expected a finite decimal, got %q", text)
	}
	return text, nil
}

type Presentation struct {
	Text  string
	Fixed func(int) string
}

func (p Presentation) Format(state fmt.State, verb rune) {
	text := p.Text
	switch verb {
	case 'f':
		if places, ok := state.Precision(); ok {
			text = p.Fixed(places)
		}
	case 'v', 's':
	default:
		fmt.Fprintf(state, "%%!%c(decimal=%s)", verb, text)
		return
	}
	if state.Flag('+') && !strings.HasPrefix(text, "-") {
		text = "+" + text
	} else if state.Flag(' ') && !strings.HasPrefix(text, "-") {
		text = " " + text
	}
	if width, ok := state.Width(); ok && width > len(text) {
		pad := " "
		if state.Flag('0') && !state.Flag('-') {
			pad = "0"
		}
		padding := strings.Repeat(pad, width-len(text))
		if state.Flag('-') {
			text += padding
		} else if pad == "0" && strings.ContainsAny(text[:1], "-+ ") {
			text = text[:1] + padding + text[1:]
		} else {
			text = padding + text
		}
	}
	_, _ = state.Write([]byte(text))
}

func Fixed(formatter fmt.Formatter, places int) string {
	return fmt.Sprintf("%."+strconv.Itoa(places)+"f", formatter)
}
