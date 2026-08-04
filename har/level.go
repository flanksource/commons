package har

import (
	"fmt"
	"strings"
)

// Level selects what a HAR collector captures. Borrowed from
// duty/connection/common.go's Debug/Trace split: at Metadata only headers,
// query strings and timings are recorded (no bodies, so no body re-read cost);
// at Full the standard collector middleware captures bodies too.
type Level int

const (
	Disabled Level = iota
	Metadata
	Full
)

func (l Level) String() string {
	switch l {
	case Metadata:
		return "metadata"
	case Full:
		return "full"
	default:
		return "disabled"
	}
}

// ParseLevel maps a property value onto a Level. "debug"/"trace" are accepted
// as synonyms for metadata/full, matching the log.level.*.har vocabulary duty
// and commons-db use. An empty string yields def; anything unrecognised is an
// error, so a typo turns into a startup failure rather than silently capturing
// the wrong thing.
func ParseLevel(value string, def Level) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return def, nil
	case "disabled", "off", "none":
		return Disabled, nil
	case "metadata", "debug":
		return Metadata, nil
	case "full", "trace", "bodies":
		return Full, nil
	default:
		return def, fmt.Errorf("invalid HAR level %q: expected metadata, full or disabled", value)
	}
}
