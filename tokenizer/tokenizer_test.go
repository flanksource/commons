package tokenizer

import (
	"fmt"
	"testing"
	"time"
)

func TestFingerprint(t *testing.T) {
	testCases := []struct {
		name   string
		inputs []map[string]any
	}{
		{
			name: "dedupe by number",
			inputs: []map[string]any{
				{
					"user": "1",
				},
				{
					"user": "2",
				},
				{
					"user": "3",
				},
			},
		},
		{
			name: "dedupe by timestamps",
			inputs: []map[string]any{
				{
					"log": fmt.Sprintf("Request received at %s", time.Now().Format(time.RFC3339)),
				},
				{
					"log": fmt.Sprintf("Request received at %s", time.Now().Add(time.Second).Format(time.RFC3339)),
				},
				{
					"log": fmt.Sprintf("Request received at %s", time.Now().Add(time.Second*2).Format(time.RFC3339)),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var expected string
			for _, input := range testCase.inputs {
				hash := TokenizeMap(input)
				if expected == "" {
					expected = hash
				} else if expected != hash {
					t.Errorf("expected %s, got %s", expected, hash)
				}
			}
		})
	}
}
