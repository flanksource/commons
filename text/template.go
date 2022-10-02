package text

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	gotemplate "text/template"

	"github.com/Masterminds/sprig"
	"github.com/dustin/go-humanize"
	"github.com/ghodss/yaml"

	"github.com/flanksource/commons/files"
	"github.com/flanksource/gomplate/v3"
)

// ToFile saves text as a temp file with an extension
func ToFile(text string, ext string) string {
	tmp := files.TempFileName("", ext)
	os.WriteFile(tmp, []byte(text), 0644)
	return tmp
}

// Template templates out a template using gomplate
func Template(template string, vars interface{}) (string, error) {
	tpl := gotemplate.New("")

	tpl, err := tpl.Funcs(GetTemplateFuncs()).Parse(template)

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

	tpl, err := tpl.Funcs(GetTemplateFuncs()).Parse(template)

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

func GetTemplateFuncs() gotemplate.FuncMap {
	funcs := gomplate.Funcs(nil)
	funcs["humanizeBytes"] = HumanizeBytes
	funcs["humanizeTime"] = humanize.Time
	funcs["humanizeDuration"] = HumanizeDuration
	funcs["ftoa"] = humanize.Ftoa
	sprigFuncs := sprig.TxtFuncMap()
	for funcName, _ := range sprigFuncs {
		if _, ok := funcs[funcName]; !ok {
			funcs[funcName] = sprigFuncs[funcName]
		}
	}
	return funcs
}
