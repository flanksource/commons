package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	mockTest = "test"
	mockENV  = "TEST"
)

var testStrings = []string{"test"}

func Test_InterpolateString_Success(t *testing.T) {
	resp := InterpolateStrings(testStrings, "123")

	assert.Equal(t, resp, testStrings)
}

func Test_Interpolate_Success(t *testing.T) {
	resp := Interpolate("test", "123")

	assert.Equal(t, resp, "test")
}

func Test_RandomString(t *testing.T) {
	testLength := 10
	resp := RandomString(testLength)

	assert.Equal(t, len(resp), testLength)

}

func Test_ShortTimeStamp(t *testing.T) {
	resp := ShortTimestamp()

	assert.NotNil(t, resp)
}

func TestGetenv_Or_Default(t *testing.T) {
	fakeEnv := NewFakeEnv()
	fakeEnv.Setenv(mockENV, mockTest)

	resp := GetEnvOrDefault(&fakeEnv, mockENV)
	assert.Equal(t, resp, mockTest)

}

type FakeEnv struct {
	values map[string]string
}

func (fk *FakeEnv) Getenv(name string) string {
	return fk.values[name]
}

func (fk *FakeEnv) Setenv(key string, value string) {
	fk.values[key] = value
}

func NewFakeEnv() FakeEnv {
	f := FakeEnv{}
	f.values = make(map[string]string)
	return f
}
