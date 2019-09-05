package text

import (
	"github.com/moshloop/commons/deps"
	"github.com/moshloop/commons/files"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

var gomplate = deps.Binary("gomplate", "", ".bin")

// ToFile saves text as a temp file with an extension
func ToFile(text string, ext string) string {
	tmp := files.TempFileName("", ext)
	ioutil.WriteFile(tmp, []byte(text), 0644)
	return tmp
}

// TemplateDir templates out a directory using gomplate
func TemplateDir(dir string, dst string, vars interface{}) error {
	data, _ := yaml.Marshal(vars)
	tmp := ToFile(string(data), "yml")
	defer os.Remove(tmp)
	return gomplate("--input-dir \"%s\" --output-dir %s -c \".=%s\"", dir, dst, tmp)
}

// Template templates out a template using gomplate
func Template(template string, vars interface{}) (string, error) {
	data, _ := yaml.Marshal(vars)
	tmp := ToFile(string(data), ".yml")
	defer os.Remove(tmp)

	in := ToFile(string(template), ".tmpl")
	defer os.Remove(in)

	out := files.TempFileName("", ".out")
	defer os.Remove(out)

	if err := gomplate("-f \"%s\" -o \"%s\" -c \".=%s\"", in, out, tmp); err != nil {
		return "", nil
	}
	dataOut, err := ioutil.ReadFile(out)
	if err != nil {
		return "", err
	}
	return string(dataOut), nil
}
