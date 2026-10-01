package properties

import (
	"fmt"
	"strconv"
	"strings"
)

// byteSizeUnits maps a normalised (lower-case) size suffix to its multiplier.
// SI suffixes are powers of 1000 and IEC suffixes are powers of 1024, matching
// the convention used by Kubernetes resource quantities, so "1MB" is 1000000
// and "1MiB" is 1048576.
var byteSizeUnits = map[string]int64{
	"":    1,
	"b":   1,
	"k":   1000,
	"kb":  1000,
	"ki":  1 << 10,
	"kib": 1 << 10,
	"m":   1000 * 1000,
	"mb":  1000 * 1000,
	"mi":  1 << 20,
	"mib": 1 << 20,
	"g":   1000 * 1000 * 1000,
	"gb":  1000 * 1000 * 1000,
	"gi":  1 << 30,
	"gib": 1 << 30,
	"t":   1000 * 1000 * 1000 * 1000,
	"tb":  1000 * 1000 * 1000 * 1000,
	"ti":  1 << 40,
	"tib": 1 << 40,
}

// ParseBytes parses a byte-size value: a decimal integer with an optional,
// case-insensitive size suffix. Accepted suffixes are B, KB/MB/GB/TB (powers
// of 1000) and KiB/MiB/GiB/TiB (powers of 1024); the bare forms K/M/G/T and
// Ki/Mi/Gi/Ti are accepted too. Whitespace between the number and the suffix
// is ignored, so "1MiB", "1 MiB" and "1048576" all parse to 1048576.
func ParseBytes(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)

	end := len(trimmed)
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			continue
		}
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		end = i
		break
	}

	number, err := strconv.ParseInt(trimmed[:end], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse byte size %q: %w", value, err)
	}

	suffix := strings.TrimSpace(trimmed[end:])
	multiplier, ok := byteSizeUnits[strings.ToLower(suffix)]
	if !ok {
		return 0, fmt.Errorf("parse byte size %q: unknown unit %q", value, suffix)
	}

	scaled := number * multiplier
	if number != 0 && scaled/number != multiplier {
		return 0, fmt.Errorf("parse byte size %q: value overflows int64", value)
	}
	return scaled, nil
}
