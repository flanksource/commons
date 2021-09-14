package text

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	gotemplate "text/template"

	"github.com/dustin/go-humanize"
	"github.com/flanksource/commons/files"
	"github.com/hairyhenderson/gomplate/v3"
	"gopkg.in/flanksource/yaml.v3"
)

// ToFile saves text as a temp file with an extension
func ToFile(text string, ext string) string {
	tmp := files.TempFileName("", ext)
	ioutil.WriteFile(tmp, []byte(text), 0644)
	return tmp
}

// Template templates out a template using gomplate
func Template(template string, vars interface{}) (string, error) {
	tpl := gotemplate.New("")

	funcs := gomplate.Funcs(nil)
	funcs["humanizeBytes"] = HumanizeBytes
	funcs["humanizeTime"] = humanize.Time
	funcs["ftoa"] = humanize.Ftoa
	tpl, err := tpl.Funcs(funcs).Parse(template)

	if err != nil {
		return "", fmt.Errorf("invalid template %s: %v", strings.Split(template, "\n")[0], err)
	}

	data, _ := yaml.Marshal(vars)
	unstructured := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &unstructured); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, unstructured); err != nil {
		return "", fmt.Errorf("error executing template %s: %v", strings.Split(template, "\n")[0], err)
	}
	return buf.String(), nil
}

// TemplateWithDelims templates out a template using gomplate using the given opening and closing Delims
func TemplateWithDelims(template, openingDelims, closingDelims string, vars interface{}) (string, error) {
	tpl := gotemplate.New("").Delims(openingDelims, closingDelims)
	funcs := gomplate.Funcs(nil)
	funcs["humanizeBytes"] = HumanizeBytes
	funcs["humanizeTime"] = humanize.Time
	funcs["ftoa"] = humanize.Ftoa
	tpl, err := tpl.Funcs(funcs).Parse(template)

	if err != nil {
		return "", fmt.Errorf("invalid template %s: %v", strings.Split(template, "\n")[0], err)
	}

	data, _ := yaml.Marshal(vars)
	unstructured := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &unstructured); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, unstructured); err != nil {
		return "", fmt.Errorf("error executing template %s: %v", strings.Split(template, "\n")[0], err)
	}
	return buf.String(), nil
}
