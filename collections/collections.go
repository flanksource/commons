package collections

import (
	"fmt"
	"reflect"
)

func takeSliceArg(arg interface{}) (out []interface{}, ok bool) {
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Slice {
		return nil, false
	}

	c := val.Len()
	out = make([]interface{}, c)
	for i := 0; i < val.Len(); i++ {
		out[i] = val.Index(i).Interface()
	}
	return out, true
}

// ToString takes an object and tries to convert it to a string
func ToString(i interface{}) string {
	if slice, ok := takeSliceArg(i); ok {
		s := ""
		for _, v := range slice {
			if s != "" {
				s += ", "
			}
			s += ToString(v)
		}
		return s

	}
	switch v := i.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case interface{}:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	default:
		// panic(fmt.Sprintf("I don't know about type %T!\n", v))
	}
	return ""
}
