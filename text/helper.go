package text

import (
	"io"
)

func SafeRead(r io.Reader) string {
	data, _ := io.ReadAll(r)
	return string(data)
}
