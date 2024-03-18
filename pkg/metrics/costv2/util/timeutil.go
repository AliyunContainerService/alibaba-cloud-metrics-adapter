package costv2

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// SecsPerMin expresses the amount of seconds in a minute
	SecsPerMin = 60.0

	// SecsPerHour expresses the amount of seconds in a minute
	SecsPerHour = 3600.0

	// SecsPerDay expressed the amount of seconds in a day
	SecsPerDay = 86400.0

	// MinsPerHour expresses the amount of minutes in an hour
	MinsPerHour = 60.0

	// MinsPerDay expresses the amount of minutes in a day
	MinsPerDay = 1440.0

	// HoursPerDay expresses the amount of hours in a day
	HoursPerDay = 24.0

	// HoursPerMonth expresses the amount of hours in a month
	HoursPerMonth = 730.0

	// DaysPerMonth expresses the amount of days in a month
	DaysPerMonth = 30.42

	// Day expresses 24 hours
	Day = time.Hour * 24.0

	Week = Day * 7.0
)

// DurationString converts a duration to a Prometheus-compatible string in
// terms of days, hours, minutes, or seconds.
func DurationString(duration time.Duration) string {
	durSecs := int64(duration.Seconds())

	durStr := ""
	if durSecs > 0 {
		if durSecs%SecsPerDay == 0 {
			// convert to days
			durStr = fmt.Sprintf("%dd", durSecs/SecsPerDay)
		} else if durSecs%SecsPerHour == 0 {
			// convert to hours
			durStr = fmt.Sprintf("%dh", durSecs/SecsPerHour)
		} else if durSecs%SecsPerMin == 0 {
			// convert to mins
			durStr = fmt.Sprintf("%dm", durSecs/SecsPerMin)
		} else if durSecs > 0 {
			// default to secs, as long as duration is positive
			durStr = fmt.Sprintf("%ds", durSecs)
		}
	}

	return durStr
}

// ParseDuration parses a duration string.
// A duration string is a possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d"
func ParseDuration(duration string) (time.Duration, error) {
	duration = CleanDurationString(duration)
	return goParseDuration(duration)
}

// CleanDurationString removes prometheus formatted prefix "offset " allong with leading a trailing whitespace
// from duration string, leaving behind a string with format [0-9+](s|m|d|h)
func CleanDurationString(duration string) string {
	duration = strings.TrimSpace(duration)
	duration = strings.TrimPrefix(duration, "offset ")
	return duration
}

// unitMap contains a list of units that can be parsed by ParseDuration
var unitMap = map[string]int64{
	"ns": int64(time.Nanosecond),
	"us": int64(time.Microsecond),
	"µs": int64(time.Microsecond), // U+00B5 = micro symbol
	"μs": int64(time.Microsecond), // U+03BC = Greek letter mu
	"ms": int64(time.Millisecond),
	"s":  int64(time.Second),
	"m":  int64(time.Minute),
	"h":  int64(time.Hour),
	"d":  int64(Day),
	"w":  int64(Week),
}

// goParseDuration is time.ParseDuration lifted from the go std library and enhanced with the ability to
// handle the "d" (day) unit. The contents of the function itself are identical to the std library, it is
// only the unitMap above that contains the added unit.
func goParseDuration(s string) (time.Duration, error) {
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
		return 0, errors.New("time: invalid duration " + quote(orig))
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error

		// The next character must be [0-9.]
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, errors.New("time: invalid duration " + quote(orig))
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
			return 0, errors.New("time: missing unit in duration " + quote(orig))
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[u]
		if !ok {
			return 0, errors.New("time: unknown unit " + quote(u) + " in duration " + quote(orig))
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, errors.New("time: invalid duration " + quote(orig))
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, errors.New("time: invalid duration " + quote(orig))
		}
	}

	if neg {
		d = -d
	}

	return time.Duration(d), nil
}

const (
	lowerhex  = "0123456789abcdef"
	runeSelf  = 0x80
	runeError = '\uFFFD'
)

// quote is lifted from the go std library to support the custom ParseDuration enhancement
func quote(s string) string {
	buf := make([]byte, 1, len(s)+2) // slice will be at least len(s) + quotes
	buf[0] = '"'
	for i, c := range s {
		if c >= runeSelf || c < ' ' {
			// This means you are asking us to parse a time.Duration or
			// time.Location with unprintable or non-ASCII characters in it.
			// We don't expect to hit this case very often. We could try to
			// reproduce strconv.Quote's behavior with full fidelity but
			// given how rarely we expect to hit these edge cases, speed and
			// conciseness are better.
			var width int
			if c == runeError {
				width = 1
				if i+2 < len(s) && s[i:i+3] == string(runeError) {
					width = 3
				}
			} else {
				width = len(string(c))
			}
			for j := 0; j < width; j++ {
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[i+j]>>4])
				buf = append(buf, lowerhex[s[i+j]&0xF])
			}
		} else {
			if c == '"' || c == '\\' {
				buf = append(buf, '\\')
			}
			buf = append(buf, string(c)...)
		}
	}
	buf = append(buf, '"')
	return string(buf)
}

var errLeadingInt = errors.New("time: bad [0-9]*") // never printed

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

// leadingFraction consumes the leading [0-9]* from s.
// It is used only for fractions, so does not return an error on overflow,
// it just stops accumulating precision.
func leadingFraction(s string) (x int64, scale float64, rem string) {
	i := 0
	scale = 1
	overflow := false
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			// It's possible for overflow to give a positive number, so take care.
			overflow = true
			continue
		}
		y := x*10 + int64(c) - '0'
		if y < 0 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}
