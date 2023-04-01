package template

import (
	"encoding/json"

	"github.com/flanksource/commons/logger"
	"github.com/tidwall/gjson"
)

func (f *Functions) JSONPath(object interface{}, jsonpath string) string {
	jsonObject, err := json.Marshal(object)
	if err != nil {
		logger.Errorf("failed to encode json: %v", err)
		return ""
	}
	value := gjson.Get(string(jsonObject), jsonpath)
	return value.String()
}
