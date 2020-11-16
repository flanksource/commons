package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"math/big"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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

// RandomStringGenerator allows generating a random string of characters from a chosen alphabet.
type RandomStringGenerator struct {
	randReader *bufio.Reader
	alphabet   []rune
}

// NewRandomStringGenerator returns a new RandomStringGenerator using the specified alphabet.
func NewRandomStringGenerator(alphabet string) (sg *RandomStringGenerator) {
	return &RandomStringGenerator{
		randReader: bufio.NewReader(rand.Reader),
		alphabet:   []rune(alphabet),
	}
}

func (rsg *RandomStringGenerator) Read(result []byte) (n int, err error) {
	r, err := rsg.RandomString(len(result))
	if err != nil {
		return 0, err
	}

	for k := range result {
		result[k] = r[k]
	}
	return len(result), nil
}

// RandomString returns a randomly generated string using the prespecified alphabet.
func (rsg *RandomStringGenerator) RandomString(length int) (result string, err error) {
	tempResult := make([]rune, length)
	for k := range tempResult {
		i, err := rand.Int(rsg.randReader, big.NewInt(int64(len(rsg.alphabet))))
		if err != nil {
			return "", err
		}
		tempResult[k] = rsg.alphabet[int(i.Int64())]
	}
	return string(tempResult), nil
}

// randomChars defines the alphabet for DefaultStringGenerator
const randomChars = "0123456789abcdefghijklmnopqrstuvwxyz"

// DefaultRandomStringGenerator is a default instance of RandomStringGenerator using the following alphabet: "0123456789abcdefghijklmnopqrstuvwxyz"
var DefaultRandomStringGenerator = NewRandomStringGenerator(randomChars)

// RandomString returns a random string generated using DefaultStringGenerator. Shorthand for "DefaultStringGenerator.RandomString"
func RandomString(length int) string {
	result, _ := DefaultRandomStringGenerator.RandomString(length)
	return result
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
