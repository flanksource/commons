/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Ported from k8s.io/apimachinery v0.36.2 pkg/util/mergepatch/errors.go; apply half only.

package mergepatch

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrBadJSONDoc                           = errors.New("invalid JSON document")
	ErrNoListOfLists                        = errors.New("lists of lists are not supported")
	ErrBadPatchFormatForPrimitiveList       = errors.New("invalid patch format of primitive list")
	ErrBadPatchFormatForRetainKeys          = errors.New("invalid patch format of retainKeys")
	ErrBadPatchFormatForSetElementOrderList = errors.New("invalid patch format of setElementOrder list")
	ErrUnsupportedStrategicMergePatchFormat = errors.New("strategic merge patch format is not supported")
)

func ErrNoMergeKey(m map[string]interface{}, k string) error {
	return fmt.Errorf("map: %v does not contain declared merge key: %s", m, k)
}

func ErrBadArgType(expected, actual interface{}) error {
	return fmt.Errorf("expected a %s, but received a %s",
		reflect.TypeOf(expected),
		reflect.TypeOf(actual))
}

func ErrBadArgKind(expected, actual interface{}) error {
	var expectedKindString, actualKindString string
	if expected == nil {
		expectedKindString = "nil"
	} else {
		expectedKindString = reflect.TypeOf(expected).Kind().String()
	}
	if actual == nil {
		actualKindString = "nil"
	} else {
		actualKindString = reflect.TypeOf(actual).Kind().String()
	}
	return fmt.Errorf("expected a %s, but received a %s", expectedKindString, actualKindString)
}

func ErrBadPatchType(t interface{}, m map[string]interface{}) error {
	return fmt.Errorf("unknown patch type: %s in map: %v", t, m)
}
