package timeinterval

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func unmarshalRangeJSON(in []byte) func(any) error {
	return func(target any) error {
		var token json.RawMessage
		if err := json.Unmarshal(in, &token); err != nil {
			return err
		}
		if len(token) > 0 && token[0] == '"' {
			return json.Unmarshal(token, target)
		}
		if string(token) == "null" {
			return fmt.Errorf("interval range must be a string or integer")
		}
		var number int
		if err := json.Unmarshal(token, &number); err != nil {
			return err
		}
		value, ok := target.(*string)
		if !ok {
			return fmt.Errorf("interval range expected *string, got %T", target)
		}
		*value = strconv.Itoa(number)
		return nil
	}
}
