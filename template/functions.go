package template

import (
	"text/template"

	"github.com/flanksource/commons/text"
	"github.com/flanksource/gomplate/v3"
)

type Functions struct {
	RightDelim, LeftDelim string
	Custom                template.FuncMap
}

func NewFunctions() *Functions {
	return &Functions{}
}

func (f *Functions) FuncMap() template.FuncMap {
	fm := gomplate.Funcs(nil)
	fm["jsonPath"] = f.JSONPath
	for k, v := range f.Custom {
		fm[k] = v
	}
	commonFuncs := text.GetTemplateFuncs()
	for funcName := range commonFuncs {
		if _, ok := fm[funcName]; !ok {
			fm[funcName] = commonFuncs[funcName]
		}
	}
	return fm
}
