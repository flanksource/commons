package tokenizer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/samber/lo"
)

type replacement struct {
	Value string
	Regex *regexp.Regexp
}

type replacements []replacement

var tokenizer replacements

func init() {
	tokenizer = newReplacements(
		"UUID", `\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b`,
		"TIMESTAMP", `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`,
		"DURATION", `\s+\d+(.\d+){0,1}(ms|s|h|d|m)`,
		"SHA256", `[a-z0-9]{64}`,
		"NUMBER", `^\d+$`,
		"HEX16", `[0-9a-f]{16}`, // matches a 16 character long hex string
	)
}

func newReplacements(pairs ...string) replacements {
	var r replacements
	for i := 0; i < len(pairs)-1; i = i + 2 {
		r = append(r, replacement{
			Value: pairs[i],
			Regex: regexp.MustCompile(pairs[i+1]),
		})
	}
	return r
}

func (replacements replacements) Tokenize(data any) string {
	switch v := data.(type) {

	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		return "0"
	case time.Duration:
		return "DURATION"
	case time.Time:
		return "TIMESTAMP"
	case string:
		out := v
		for _, r := range replacements {
			out = r.Regex.ReplaceAllString(out, r.Value)
			if out == r.Value {
				break
			}
		}
		return out
	}

	return fmt.Sprintf("%v", data)
}

func TokenizeMap(data map[string]any) string {
	out := make(map[string]any, len(data))
	for k, v := range data {
		out[k] = tokenizer.Tokenize(v)
	}

	return hash(out)
}

func Tokenize(input string) string {
	return tokenizer.Tokenize(input)
}

func hash(data map[string]any) string {
	keys := lo.Keys(data)
	sort.Strings(keys)

	h := md5.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(data[k].(string)))
	}

	return hex.EncodeToString(h.Sum(nil)[:])
}
