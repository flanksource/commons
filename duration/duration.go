// MIT License
// Copyright Wijnand Modderman-Lenstra (maze.io)

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// FORKED from https://git.maze.io/go/duration/src/branch/master/duration.go

// Package duration provides enhanced duration parsing and formatting with support
// for extended time units beyond the standard library.
//
// This package extends Go's standard time.Duration to support larger units like
// days, weeks, months, and years, making it more suitable for human-readable
// time representations and configuration files.
//
// Key features:
//   - Parse durations with extended units: "d" (days), "w" (weeks), "y" (years)
//   - Human-readable string formatting: "3d12h" instead of "84h"
//   - Automatic precision adjustment based on magnitude
//   - Compatible with standard time.Duration
//
// Supported time units:
//   - ns: nanoseconds
//   - us/µs/μs: microseconds
//   - ms: milliseconds
//   - s: seconds
//   - m: minutes
//   - h: hours
//   - d: days (24 hours)
//   - w: weeks (7 days)
//   - y: years (365 days, approximation)
//
// Basic usage:
//
//	// Parse duration strings
//	d, err := duration.ParseDuration("3d12h30m")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(d) // Output: 3d12h30m
//
//	// Create from standard units
//	d = duration.Day * 7 + duration.Hour * 6
//	fmt.Println(d) // Output: 1w6h
//
//	// Convert to standard time.Duration
//	td := time.Duration(d)
//
//	// Get components
//	fmt.Printf("Hours: %.2f, Days: %.2f\n", d.Hours(), d.Days())
//
// The formatting automatically adjusts precision based on the duration magnitude:
//   - Durations over a week: formatted as weeks, days, hours (1w2d3h)
//   - Durations over a day: formatted as days, hours, minutes (2d3h45m)
//   - Smaller durations: include seconds and sub-second precision
package duration

import (
	"errors"
	"strings"
	"time"
)

// Duration represents a time duration with support for extended units.
// It embeds time.Duration and can be used anywhere a time.Duration is expected,
// but provides enhanced parsing and formatting capabilities.
type Duration time.Duration

// String returns a string representing the duration in the form "3d1h3m".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. Duration more than a day or more than a
// week lose granularity and are truncated to resp. days-hours-minutes and
// weeks-days-hours. The zero duration formats as 0s.
func (d Duration) String() string {

	if d.Hours() > 24*7 {
		d = d - d%Duration(time.Hour)
	} else if d.Hours() > 24 {
		d = d - d%Duration(time.Minute)
	} else if d.Minutes() > 2 {
		d = d - d%Duration(time.Second)
	} else if d.Seconds() > 2 {
		d = d - d%Duration(time.Millisecond)
	} else if d.Nanoseconds() > 2*1000*1000 {
		d = d - d%Duration(time.Microsecond)
	}
	// Largest time is 2540400h10m10.000000000s
	var buf [32]byte
	w := len(buf)

	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}

	if u < uint64(Second) {
		// Special case: if duration is smaller than a second,
		// use smaller units, like 1.2ms
		var prec int
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			return "0s"
		case u < uint64(Microsecond):
			// print nanoseconds
			prec = 0
			buf[w] = 'n'
		case u < uint64(Millisecond):
			// print microseconds
			prec = 3
			// U+00B5 'µ' micro sign == 0xC2 0xB5
			w-- // Need room for two bytes.
			copy(buf[w:], "µ")
		default:
			// print milliseconds
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)

	} else if u > uint64(Week) {
		// Special case: if duration is larger than a week,
		// use bigger units like 4w3d2h
		w--
		buf[w] = 'h'

		u /= uint64(Hour)

		// u is now integer hours
		w = fmtInt(buf[:w], u%24)
		u /= 24

		// u is now integer days
		if u > 0 {
			w--
			buf[w] = 'd'
			w = fmtInt(buf[:w], u%7)
			u /= 7

			// u is now integer weeks
			// Stop at hours because days can be different lengths.
			if u > 0 {
				w--
				buf[w] = 'w'
				w = fmtInt(buf[:w], u)
			}
		}

	} else if u > uint64(Day) {
		// Special case: if duration is larger than a day,
		// use bigger units like 3d2h6m
		w--
		buf[w] = 'm'

		u /= uint64(Minute)

		// u is now integer minutes
		w = fmtInt(buf[:w], u%60)
		u /= 60

		// u is now integer hours
		if u > 0 {
			w--
			buf[w] = 'h'
			w = fmtInt(buf[:w], u%24)
			u /= 24

			// u is now integer weeks
			if u > 0 {
				w--
				buf[w] = 'd'
				w = fmtInt(buf[:w], u)
			}
		}

	} else {
		w--
		buf[w] = 's'

		w, u = fmtFrac(buf[:w], u, 9)

		// u is now integer seconds
		w = fmtInt(buf[:w], u%60)
		u /= 60

		// u is now integer minutes
		if u > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], u%60)
			u /= 60

			// u is now integer hours
			// Stop at hours because days can be different lengths.
			if u > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], u)
			}
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}

	return strings.ReplaceAll(strings.ReplaceAll(string(buf[w:]), "0s", ""), "0m", "")
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros.  it omits the decimal
// point too when the fraction is 0.  It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

// Nanoseconds returns the duration as an integer nanosecond count.
func (d Duration) Nanoseconds() int64 { return int64(d) }

// Seconds returns the duration as a floating point number of seconds.
func (d Duration) Seconds() float64 {
	sec := d / Second
	nsec := d % Second
	return float64(sec) + float64(nsec)*1e-9
}

// Hours returns the duration as a floating point number of hours.
func (d Duration) Hours() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)*(1e-9/60/60)
}

// Days returns the duration as a floating point number of days.
func (d Duration) Days() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)*(1e-9/60/60/24)
}

// Weeks returns the duration as a floating point number of days.
func (d Duration) Weeks() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)*(1e-9/60/60/24/7)
}

// Minutes returns the duration as a floating point number of minutes.
func (d Duration) Minutes() float64 {
	min := d / Minute
	nsec := d % Minute
	return float64(min) + float64(nsec)*(1e-9/60)
}

// Standard unit of time.
var (
	Nanosecond  = Duration(time.Nanosecond)
	Microsecond = Duration(time.Microsecond)
	Millisecond = Duration(time.Millisecond)
	Second      = Duration(time.Second)
	Minute      = Duration(time.Minute)
	Hour        = Duration(time.Hour)
	Day         = Hour * 24
	Week        = Day * 7
	Fortnight   = Week * 2
	Month       = Day * 30    // Approximation
	Year        = Day * 365   // Approximation
	Decade      = Year * 10   // Approximation
	Century     = Year * 100  // Approximation
	Millennium  = Year * 1000 // Approximation
)

var errLeadingInt = errors.New("duration: bad [0-9]*") // never printed
// leadingInt consumes the leading [0-9]* from s.
func leadingInt(s string) (x int64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", errLeadingInt
		}
		x = x*10 + int64(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", errLeadingInt
		}
	}
	return x, s[i:], nil
}

var unitMap = map[string]int64{
	"ns": int64(Nanosecond),
	"us": int64(Microsecond),
	"µs": int64(Microsecond), // U+00B5 = micro symbol
	"μs": int64(Microsecond), // U+03BC = Greek letter mu
	"ms": int64(Millisecond),
	"s":  int64(Second),
	"m":  int64(Minute),
	"h":  int64(Hour),
	"d":  int64(Day),
	"w":  int64(Week),
	"y":  int64(Year), // Approximation
}

// ParseDuration parses a duration string with support for extended time units.
// A duration string is a possibly signed sequence of decimal numbers, each with
// optional fraction and a unit suffix, such as "300ms", "-1.5h", "2h45m", or "3d12h".
//
// Valid time units are:
//   - "ns": nanoseconds
//   - "us" (or "µs"/"μs"): microseconds
//   - "ms": milliseconds
//   - "s": seconds
//   - "m": minutes
//   - "h": hours
//   - "d": days (24 hours)
//   - "w": weeks (7 days)
//   - "y": years (365 days)
//
// Examples:
//
//	ParseDuration("2h30m")      // 2 hours 30 minutes
//	ParseDuration("1.5d")       // 1.5 days (36 hours)
//	ParseDuration("3w2d")       // 3 weeks 2 days
//	ParseDuration("-30m")       // negative 30 minutes
//	ParseDuration("1d12h30m")   // 1 day 12 hours 30 minutes
func ParseDuration(s string) (Duration, error) {
	// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
	orig := s
	var d int64
	neg := false

	// Consume [-+]?
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	// Special case: if all that is left is "0", this is zero.
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("time: invalid duration " + orig)
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error

		// The next character must be [0-9.]
		if s[0] != '.' && (s[0] < '0' || s[0] > '9') {
			return 0, errors.New("time: invalid duration " + orig)
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, errors.New("time: invalid duration " + orig)
		}
		pre := pl != len(s) // whether we consumed anything before a period
		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, s, err = leadingInt(s)
			if err != nil {
				return 0, errors.New("time: invalid duration " + orig)
			}
			for n := pl - len(s); n > 0; n-- {
				scale *= 10
			}
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, errors.New("time: invalid duration " + orig)
		}

		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, errors.New("time: missing unit in duration " + orig)
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[u]
		if !ok {
			return 0, errors.New("time: unknown unit " + u + " in duration " + orig)
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, errors.New("time: invalid duration " + orig)
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, errors.New("time: invalid duration " + orig)
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, errors.New("time: invalid duration " + orig)
		}
	}

	if neg {
		d = -d
	}
	return Duration(d), nil
}
