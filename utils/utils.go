package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"html/template"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type environment interface {
	Getenv(name string) string
	Setenv(name, value string)
}

type osEnv struct{}

func (o *osEnv) Getenv(name string) string {
	return os.Getenv(name)
}
func (o *osEnv) Setenv(name, value string) {
	os.Setenv(name, value)
}

// GetEnvOrDefault returns the first non-empty environment variable
func GetEnvOrDefault(env environment, names ...string) string {
	for _, name := range names {
		if val := env.Getenv(name); val != "" {
			return val
		}
	}
	return ""
}

// ShortTimestamp returns a shortened timestamp using
// week of year + day of week to represent a day of the
// e.g. 1st of Jan on a Tuesday is 13
func ShortTimestamp() string {
	_, week := time.Now().ISOWeek()
	return fmt.Sprintf("%d%d-%s", week, time.Now().Weekday(), time.Now().Format("150405"))
}

// randomChars defines the alphanumeric characters that can be part of a random string
const randomChars = "0123456789abcdefghijklmnopqrstuvwxyz"

// RandomString returns a random string consisting of the characters in
// randomChars, with the length customized by the parameter
func RandomString(length int) string {
	// len("0123456789abcdefghijklmnopqrstuvwxyz") = 36 which doesn't evenly divide
	// the possible values of a byte: 256 mod 36 = 4. Discard any random bytes we
	// read that are >= 252 so the bytes we evenly divide the character set.
	const maxByteValue = 252

	var (
		b     byte
		err   error
		token = make([]byte, length)
	)

	reader := bufio.NewReaderSize(rand.Reader, length*2)
	for i := range token {
		for {
			if b, err = reader.ReadByte(); err != nil {
				return ""
			}
			if b < maxByteValue {
				break
			}
		}

		token[i] = randomChars[int(b)%len(randomChars)]
	}

	return string(token)
}

// Interpolate templatises the string using the vars as the context
func Interpolate(arg string, vars interface{}) string {
	tmpl, err := template.New("test").Parse(arg)
	if err != nil {
		log.Errorf("Failed to parse template %s -> %s\n", arg, err)
		return arg
	}
	buf := bytes.NewBufferString("")

	err = tmpl.Execute(buf, vars)
	if err != nil {
		log.Errorf("Failed to execute template %s -> %s\n", arg, err)
		return arg
	}
	return buf.String()

}

// InterpolateStrings templatises each string in the slice using the vars as the context
func InterpolateStrings(arg []string, vars interface{}) []string {
	out := make([]string, len(arg))
	for i, e := range arg {
		out[i] = Interpolate(e, vars)
	}
	return out
}
