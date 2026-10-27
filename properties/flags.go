// Portions adapted from github.com/spf13/pflag/string_to_string.go.
// Copyright (c) 2012 Alex Ogier. All rights reserved.
// Copyright (c) 2012 The Go Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause
// See LICENSE-PFLAG for the license.
package properties

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
)

type flagValue interface {
	Set(string) error
	String() string
	Type() string
}

// BindFlags infers the flag value interface from VarP, preserving pflag metadata.
func BindFlags[V flagValue, F interface {
	VarP(V, string, string, string)
}](flags F) {
	value, ok := any(&commandlineFlag{parsed: make(map[string]string)}).(V)
	if !ok {
		panic("properties flag set requires an interface-compatible flag value")
	}
	flags.VarP(value, "properties", "P", "System properties")
}

type commandlineFlag struct{ parsed map[string]string }

func (f *commandlineFlag) Set(value string) error {
	pairs := []string{strings.Trim(value, `"`)}
	if strings.Count(value, "=") > 1 {
		var err error
		pairs, err = csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return err
		}
	}
	next := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("%s must be formatted as key=value", pair)
		}
		next[key] = value
	}
	for key, value := range next {
		f.parsed[key] = value
	}
	commandline.replace(f.parsed)
	return nil
}

func (f *commandlineFlag) Type() string { return "stringToString" }

func (f *commandlineFlag) String() string {
	keys := make([]string, 0, len(f.parsed))
	for key := range f.parsed {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	record := make([]string, 0, len(keys))
	for _, key := range keys {
		record = append(record, key+"="+f.parsed[key])
	}
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	if err := writer.Write(record); err != nil {
		panic(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		panic(err)
	}
	return "[" + strings.TrimSpace(out.String()) + "]"
}
