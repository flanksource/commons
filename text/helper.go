package text

import (
	"io"
	"io/ioutil"
)

func SafeRead(r io.Reader) string {
	data, _ := ioutil.ReadAll(r)
	return string(data)
}
