package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func Ptr[T any](value T) *T {
	return &value
}

// Coalesce returns the first non-zero element
func Coalesce[T comparable](arr ...T) T {
	var zeroVal T
	for _, item := range arr {
		if item != zeroVal {
			return item
		}
	}

	return zeroVal
}

// GetEnvOrDefault returns the first non-empty environment variable
func GetEnvOrDefault(names ...string) string {
	for _, name := range names {
		if val := os.Getenv(name); val != "" {
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

func RandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Cannot generate random data: %v", err)
		return ""
	}
	return hex.EncodeToString(bytes)
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

// NormalizeVersion appends "v" to version string if it's not exist
func NormalizeVersion(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// Stringify converts the given value to a string.
// If the value is already a string, it is returned as is.
func Stringify(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
