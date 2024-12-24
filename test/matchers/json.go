package matchers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bsm/gomega/gcustom"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/samber/lo"
)

func MatchMap(a map[string]string) gcustom.CustomGomegaMatcher {
	return gcustom.MakeMatcher(func(b map[string]string) (bool, error) {

		var err error
		actualJSONb, err := json.Marshal(a)
		if err != nil {
			return false, err
		}

		expectedJSONb, err := json.Marshal(b)
		if err != nil {
			return false, err
		}

		expectedJSON, err := NormalizeJSON(string(expectedJSONb))
		if err != nil {
			return false, err
		}

		actualJSON, err := NormalizeJSON(string(actualJSONb))
		if err != nil {
			return false, err
		}

		diff, err := generateDiff(string(actualJSON), string(expectedJSON))
		if err != nil {
			return false, err
		}
		if len(diff) > 0 {
			return false, fmt.Errorf("%v", diff)
		}
		return true, nil
	})
}

// NormalizeJSON returns an indented json string.
// The keys are sorted lexicographically.
func NormalizeJSON(jsonStr string) (string, error) {
	var jsonStrMap interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonStrMap); err != nil {
		return "", err
	}

	jsonStrIndented, err := json.MarshalIndent(jsonStrMap, "", "\t")
	if err != nil {
		return "", err
	}

	return string(jsonStrIndented), nil
}

// generateDiff calculates the diff (git style) between the given 2 configs.
func generateDiff(newConf, prevConfig string) (string, error) {
	// We want a nicely indented json config with each key-vals in new line
	// because that gives us a better diff. A one-line json string config produces diff
	// that's not very helpful.
	before, err := NormalizeJSON(prevConfig)
	if err != nil {
		return "", fmt.Errorf("failed to normalize json for previous config: %w", err)
	}

	after, err := NormalizeJSON(newConf)
	if err != nil {
		return "", fmt.Errorf("failed to normalize json for new config: %w", err)
	}

	edits := myers.ComputeEdits("", before, after)
	if len(edits) == 0 {
		return "", nil
	}

	diff := fmt.Sprint(gotextdiff.ToUnified("before", "after", before, edits))
	return diff, nil
}

func MatchJson(a any) gcustom.CustomGomegaMatcher {
	return gcustom.MakeMatcher(func(b any) (bool, error) {
		return CompareObjects(a, b)
	})
}

func CompareObjects(actual, expected any) (bool, error) {
	if lo.IsNil(actual) && lo.IsNil(expected) {
		return true, nil
	}

	switch v := actual.(type) {
	case json.RawMessage:
		if len(v) == 0 && len(expected.(json.RawMessage)) == 0 {
			return true, nil
		} else {
			// Validate JSON before comparison
			if !json.Valid(v) {
				return false, fmt.Errorf("invalid JSON in actual: %s", string(v))
			}
			expectedJSON := expected.(json.RawMessage)
			if !json.Valid(expectedJSON) {
				return false, fmt.Errorf("invalid JSON in expected: %s", string(expectedJSON))
			}
			return CompareJSON(v, expectedJSON)
		}
	}

	switch v := expected.(type) {
	case json.RawMessage:
		if len(v) == 0 && len(actual.(json.RawMessage)) == 0 {
			return true, nil
		} else {
			// Validate JSON before comparison
			actualJSON := actual.(json.RawMessage)
			if !json.Valid(actualJSON) {
				return false, fmt.Errorf("invalid JSON in actual: %s", string(actualJSON))
			}
			if !json.Valid(v) {
				return false, fmt.Errorf("invalid JSON in expected: %s", string(v))
			}
			return CompareJSON(actualJSON, v)
		}
	}

	if actual == nil {
		return false, errors.New("actual is nil")
	}
	_a, err := json.MarshalIndent(actual, "", " ")
	if err != nil {
		return false, err
	}
	_b, err := json.MarshalIndent(expected, "", " ")
	if err != nil {
		return false, err
	}
	return CompareJSON(_a, _b)
}

func CompareJSON(actual []byte, expected []byte) (bool, error) {
	var valueA, valueB = actual, expected
	var err error

	diff, err := generateDiff(string(valueA), string(valueB))
	if err != nil {
		return false, err
	}
	if diff != "" {
		return false, errors.New(diff)
	}
	return true, nil
}
