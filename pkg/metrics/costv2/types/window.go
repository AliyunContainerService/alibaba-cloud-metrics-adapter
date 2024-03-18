package costv2

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"sync"
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

	WindowLayout = "20060102150405"
)

var (
	durationRegex       = regexp.MustCompile(`^(\d+)(m|h|d|w)$`)
	durationOffsetRegex = regexp.MustCompile(`^(\d+)(m|h|d|w) offset (\d+)(m|h|d|w)$`)
	offesetRegex        = regexp.MustCompile(`^(\+|-)(\d\d):(\d\d)$`)
	rfc3339             = `\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ`
	rfcRegex            = regexp.MustCompile(fmt.Sprintf(`(%s),(%s)`, rfc3339, rfc3339))
	timestampPairRegex  = regexp.MustCompile(`^(\d+)[,|-](\d+)$`)

	tOffsetLock sync.Mutex
	tOffset     *time.Duration

	utcOffsetLock sync.Mutex
	utcOffsetDur  *time.Duration
)

type Window struct {
	start *time.Time
	end   *time.Time
}

func (w Window) Start() *time.Time {
	return w.start
}

// IsNegative a Window is negative if start and end are not null and end is before start
func (w Window) IsNegative() bool {
	return !w.IsOpen() && w.end.Before(*w.Start())
}

// IsOpen a Window is open if it has a nil start or end
func (w Window) IsOpen() bool {
	return w.start == nil || w.end == nil
}

func (w Window) Duration() time.Duration {
	if w.IsOpen() {
		// TODO test
		return time.Duration(math.Inf(1.0))
	}

	return w.end.Sub(*w.start)
}

func (w Window) End() *time.Time {
	return w.end
}

func (w Window) Equal(that Window) bool {
	if w.start != nil && that.start != nil && !w.start.Equal(*that.start) {
		// starts are not nil, but not equal
		return false
	}

	if w.end != nil && that.end != nil && !w.end.Equal(*that.end) {
		// ends are not nil, but not equal
		return false
	}

	if (w.start == nil && that.start != nil) || (w.start != nil && that.start == nil) {
		// one start is nil, the other is not
		return false
	}

	if (w.end == nil && that.end != nil) || (w.end != nil && that.end == nil) {
		// one end is nil, the other is not
		return false
	}

	// either both starts are nil, or they match; likewise for the ends
	return true
}

func (w Window) GetLabelSelectorStr() string {
	start := *w.Start()
	end := *w.End()
	startStr := start.Format(WindowLayout)
	endStr := end.Format(WindowLayout)
	return fmt.Sprintf("window_start=%s,window_end=%s,window_layout=%s", startStr, endStr, WindowLayout)
}

// parseWindow generalizes the parsing of window strings, relative to a given
// moment in time, defined as "now".
func parseWindow(window string, now time.Time) (Window, error) {
	// compute UTC offset in terms of minutes
	offHr := now.UTC().Hour() - now.Hour()
	offMin := (now.UTC().Minute() - now.Minute()) + (offHr * 60)
	offset := time.Duration(offMin) * time.Minute

	if window == "today" {
		start := now
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)

		end := start.Add(time.Hour * 24)

		return NewWindow(&start, &end), nil
	}

	if window == "yesterday" {
		start := now
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		start = start.Add(time.Hour * -24)

		end := start.Add(time.Hour * 24)

		return NewWindow(&start, &end), nil
	}

	if window == "week" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 Sunday of the current week
		start = start.Add(-24 * time.Hour * time.Duration(start.Weekday()))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "lastweek" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 Sunday of last week
		start = start.Add(-24 * time.Hour * time.Duration(start.Weekday()+7))

		end := start.Add(7 * 24 * time.Hour)

		return NewWindow(&start, &end), nil
	}

	if window == "month" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 1st of this month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "month" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 1st of this month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "lastmonth" {
		// now
		end := now
		// 00:00 today, accounting for timezone offset
		end = end.Truncate(time.Hour * 24)
		end = end.Add(offset)
		// 00:00 1st of this month
		end = end.Add(-24 * time.Hour * time.Duration(end.Day()-1))

		// 00:00 last day of last month
		start := end.Add(-24 * time.Hour)
		// 00:00 1st of last month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		return NewWindow(&start, &end), nil
	}

	// Match duration strings; e.g. "45m", "24h", "7d"
	match := durationRegex.FindStringSubmatch(window)
	if match != nil {
		dur := time.Minute
		if match[2] == "h" {
			dur = time.Hour
		}
		if match[2] == "d" {
			dur = 24 * time.Hour
		}
		if match[2] == "w" {
			dur = Week
		}

		num, _ := strconv.ParseInt(match[1], 10, 64)

		end := now
		start := end.Add(-time.Duration(num) * dur)

		// when using windows such as "7d" and "1w", we have to have a definition for what "the past X days" means.
		// let "the past X days" be defined as the entirety of today plus the entirety of the past X-1 days, where
		// "entirety" is defined as midnight to midnight, UTC. given this definition, we round forward the calculated
		// start and end times to the nearest day to align with midnight boundaries
		if match[2] == "d" || match[2] == "w" {
			end = end.Truncate(Day).Add(Day)
			start = start.Truncate(Day).Add(Day)
		}

		return NewWindow(&start, &end), nil
	}

	// Match duration strings with offset; e.g. "45m offset 15m", etc.
	match = durationOffsetRegex.FindStringSubmatch(window)
	if match != nil {
		end := now

		offUnit := time.Minute
		if match[4] == "h" {
			offUnit = time.Hour
		}
		if match[4] == "d" {
			offUnit = 24 * time.Hour
		}
		if match[4] == "w" {
			offUnit = 24 * Week
		}

		offNum, _ := strconv.ParseInt(match[3], 10, 64)

		end = end.Add(-time.Duration(offNum) * offUnit)

		durUnit := time.Minute
		if match[2] == "h" {
			durUnit = time.Hour
		}
		if match[2] == "d" {
			durUnit = 24 * time.Hour
		}
		if match[2] == "w" {
			durUnit = Week
		}

		durNum, _ := strconv.ParseInt(match[1], 10, 64)

		start := end.Add(-time.Duration(durNum) * durUnit)

		return NewWindow(&start, &end), nil
	}

	// Match timestamp pairs, e.g. "1586822400,1586908800" or "1586822400-1586908800"
	match = timestampPairRegex.FindStringSubmatch(window)
	if match != nil {
		s, _ := strconv.ParseInt(match[1], 10, 64)
		e, _ := strconv.ParseInt(match[2], 10, 64)
		start := time.Unix(s, 0).UTC()
		end := time.Unix(e, 0).UTC()
		return NewWindow(&start, &end), nil
	}

	// Match RFC3339 pairs, e.g. "2020-04-01T00:00:00Z,2020-04-03T00:00:00Z"
	match = rfcRegex.FindStringSubmatch(window)
	if match != nil {
		start, _ := time.Parse(time.RFC3339, match[1])
		end, _ := time.Parse(time.RFC3339, match[2])
		return NewWindow(&start, &end), nil
	}

	return Window{nil, nil}, fmt.Errorf("illegal window: %s", window)
}

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

// ParseWindowWithOffset parses the given window string within the context of
// the timezone defined by the UTC offset.
func ParseWindowWithOffset(window string, offset time.Duration) (Window, error) {
	loc := time.FixedZone("", int(offset.Seconds()))
	now := time.Now().In(loc)
	return parseWindow(window, now)
}

// NewWindow creates and returns a new Window instance from the given times
func NewWindow(start, end *time.Time) Window {
	return Window{
		start: start,
		end:   end,
	}
}

// NewClosedWindow creates and returns a new Window instance from the given
// times, which cannot be nil, so they are value types.
func NewClosedWindow(start, end time.Time) Window {
	return Window{
		start: &start,
		end:   &end,
	}
}
