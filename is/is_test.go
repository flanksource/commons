package is

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var goodFileNames = []string{"good.zip", "good.tar.gz", "good.gz"}
var badFileNames = []string{"bad.zi", "bad.tar", "bad.g"}

const (
	badDir   = "bad/directory"
	notSlice = "notSlice"
)

func Test_Archive_Returns_True(t *testing.T) {

	for _, file := range goodFileNames {
		assert.True(t, Archive(file))
	}
}

func Test_Archive_Returns_Bad(t *testing.T) {
	for _, file := range badFileNames {
		assert.False(t, Archive(file))
	}
}

func Test_File_Returns_True(t *testing.T) {

	dir, err := os.Getwd()
	assert.NoError(t, err)

	assert.True(t, File(dir))
}

func Test_File_Returns_False(t *testing.T) {
	assert.False(t, File(badDir))
}

func Test_IsSlice_Returns_True(t *testing.T) {
	assert.True(t, Slice(goodFileNames))
}

func Test_Slice_Returns_False(t *testing.T) {
	assert.False(t, Slice(notSlice))
}
