package utils

import "testing"

func TestRandomString(t *testing.T) {
	stringCount := 5
	stringLength := 64

	// Generate stringCount random strings
	results := make([]string, stringCount)
	for k := range results {
		results[k] = RandomString(stringLength)
		if results[k] == "" {
			t.Error("Random string generation failed.")
		}
		if len(results[k]) != stringLength {
			t.Error("Random string length is incorrect.")
		}
	}

	// Check if all strings are unique
	// Yeah technically this could plausibly fail, but honestly if you get a collision on this much entropy when running tests, go buy a lotto ticket.
	for k1 := range results {
		for k2 := range results {
			if k1 != k2 && results[k1] == results[k2] {
				t.Errorf(`Randomly generated strings aren't properly random. String #%d and #%d both have the same value: "%s"`, k1, k2, results[k1])
			}
		}
	}
}
