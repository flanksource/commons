/*
Copyright (c) 2015-Present CloudFoundry.org Foundation, Inc. All Rights Reserved.

This project contains software that is Copyright (c) 2013-2015 Pivotal Software, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This project may include a number of subcomponents with separate
copyright notices and license terms. Your use of these subcomponents
is subject to the terms and conditions of each subcomponent's license,
as noted in the LICENSE file.
*/

// Package text provides utilities for text formatting, humanization, and manipulation.
//
// The package offers functions to format bytes and numbers into human-readable strings,
// handle durations, safely read from io.Reader, and indent text output. It includes
// utilities commonly needed for CLI tools and logging.
//
// Key features:
//   - Human-readable byte formatting (e.g., "10M", "12.5K")
//   - Human-readable number formatting with metric suffixes
//   - Duration parsing and formatting
//   - Text indentation utilities
//   - Safe reader utilities
//
// Byte formatting example:
//
//	size := int64(1536)
//	fmt.Println(text.HumanizeBytes(size))  // "1.5K"
//	
//	size = 1073741824
//	fmt.Println(text.HumanizeBytes(size))  // "1G"
//
// Number formatting example:
//
//	count := 1500
//	fmt.Println(text.HumanizeInt(count))   // "1.5k"
//	
//	count = 2000000
//	fmt.Println(text.HumanizeInt(count))   // "2m"
//
// Duration example:
//
//	d := 3*time.Hour + 30*time.Minute
//	fmt.Println(text.HumanizeDuration(d))  // "3h30m"
//	
//	age := text.Age(time.Now().Add(-24 * time.Hour))
//	fmt.Println(age)  // "1d"
//
// Indentation example:
//
//	indented := text.String("  ", "line1\nline2\nline3")
//	// Result:
//	//   line1
//	//   line2
//	//   line3
package text

import (
	"strconv"
	"strings"

	"github.com/flanksource/commons/logger"
)

const (
	BYTE = 1 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
	PETABYTE
	EXABYTE
)

// HumanizeBytes returns a human-readable byte string of the form 10M, 12.5K, and so forth.  
// The following units are available:
//
//	E: Exabyte  (1024^6 bytes)
//	P: Petabyte (1024^5 bytes)
//	T: Terabyte (1024^4 bytes)
//	G: Gigabyte (1024^3 bytes)
//	M: Megabyte (1024^2 bytes)
//	K: Kilobyte (1024 bytes)
//	B: Byte
//
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
// The function accepts uint, uint64, int, int64, or string as input.
//
// Examples:
//
//	HumanizeBytes(1024)        // "1K"
//	HumanizeBytes(1536)        // "1.5K"
//	HumanizeBytes(1048576)     // "1M"
//	HumanizeBytes("1073741824") // "1G"
func HumanizeBytes(size interface{}) string {
	unit := ""
	var bytes uint64
	var err error
	switch t := size.(type) {
	case uint:
		bytes = uint64(t)
	case uint64:
		bytes = t
	case int:
		bytes = uint64(t)
	case int64:
		bytes = uint64(t)
	case string:
		bytes, err = strconv.ParseUint(t, 10, 64)
		if err != nil {
			logger.Debugf("error converting string to bytes: %v", err)
			return ""
		}
	}

	value := float64(bytes)

	switch {
	case bytes >= EXABYTE:
		unit = "E"
		value = value / EXABYTE
	case bytes >= PETABYTE:
		unit = "P"
		value = value / PETABYTE
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0B"
	}

	result := strconv.FormatFloat(value, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + unit
}

const (
	KILO = 1000
	MEGA = 1000 * KILO
	GIGA = 1000 * MEGA
)

// HumanizeInt formats integers with metric suffixes for readability.
// It uses decimal (base-10) units: k for thousands, m for millions, b for billions.
//
// The function accepts uint, uint64, int, int64, or string as input.
//
// Examples:
//
//	HumanizeInt(1500)        // "1.5k"
//	HumanizeInt(2000000)     // "2m"  
//	HumanizeInt(1500000000)  // "1.5b"
//	HumanizeInt("1000")      // "1k"
func HumanizeInt(size interface{}) string {
	unit := ""
	var val uint64
	var err error
	switch t := size.(type) {
	case uint:
		val = uint64(t)
	case uint64:
		val = t
	case int:
		val = uint64(t)
	case int64:
		val = uint64(t)
	case string:
		val, err = strconv.ParseUint(t, 10, 64)
		if err != nil {
			logger.Debugf("error converting string to bytes: %v", err)
			return ""
		}
	}

	value := float64(val)

	switch {
	case val >= GIGA:
		unit = "b"
		value = value / GIGA
	case val >= MEGA:
		unit = "b"
		value = value / MEGA
	case val >= KILO:
		unit = "k"
		value = value / KILO
	case val == 0:
		return "0"
	}

	result := strconv.FormatFloat(value, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + unit
}
