package template

import (
	"bytes"
	"fmt"
	"strings"
	gotemplate "text/template"

	yaml "gopkg.in/flanksource/yaml.v3"
)

// Template templates out a template using gomplate
func (f *Functions) Template(template string, vars interface{}) (string, error) {
	if strings.TrimSpace(template) == "" {
		return "", nil
	}
	tpl := gotemplate.New("")
	if f.LeftDelim != "" {
		tpl = tpl.Delims(f.LeftDelim, f.RightDelim)
	}

	tpl, err := tpl.Funcs(f.FuncMap()).Parse(template)

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
