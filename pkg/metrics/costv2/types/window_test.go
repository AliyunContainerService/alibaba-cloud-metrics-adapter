package costv2

import (
	util "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/util"
	"testing"
	"time"
)

func TestParseWindow(t *testing.T) {
	now := time.Now()
	offHr := now.UTC().Hour() - now.Hour()
	offMin := (now.UTC().Minute() - now.Minute()) + (offHr * 60)
	offset := time.Duration(offMin) * time.Minute

	testCases := []struct {
		name     string
		window   string
		now      time.Time
		expected Window
		wantErr  bool
	}{
		{
			name:   "today window",
			window: "today",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time {
					return &t
				}(now.Truncate(24*time.Hour).Add(offset)),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "yesterday window",
			window: "yesterday",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Truncate(24*time.Hour).Add(offset).Add(-24*time.Hour),
				),
				func(t time.Time) *time.Time { return &t }(
					now.Truncate(24*time.Hour).Add(offset),
				),
			),
			wantErr: false,
		},
		{
			name:   "week window",
			window: "week",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.AddDate(0, 0, -(int(now.Weekday())+6)%7).Truncate(24*time.Hour).Add(offset),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "lastweek window",
			window: "lastweek",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.AddDate(0, 0, -(int(now.Weekday())+6)%7).Truncate(24*time.Hour).Add(offset).Add(-7*24*time.Hour),
				),
				func(t time.Time) *time.Time { return &t }(
					now.AddDate(0, 0, -(int(now.Weekday())+6)%7).Truncate(24*time.Hour).Add(offset),
				),
			),
			wantErr: false,
		},
		{
			name:   "month window",
			window: "month",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Add(-24*time.Hour*time.Duration(now.Day()-1)).Truncate(24*time.Hour).Add(offset),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "45m window",
			window: "45m",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Add(-45*time.Minute),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "24h window",
			window: "24h",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Add(-24*time.Hour),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "7d window",
			window: "7d",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Add(-7*24*time.Hour).Truncate(util.Day).Add(util.Day).Add(offset),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "1w window",
			window: "1w",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					now.Add(-7*24*time.Hour).Truncate(util.Day).Add(util.Day).Add(offset),
				),
				&now,
			),
			wantErr: false,
		},
		{
			name:   "timestamp window",
			window: "1586822400,1586908800",
			now:    now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(
					time.Unix(1586822400, 0).UTC(),
				),
				func(t time.Time) *time.Time { return &t }(
					time.Unix(1586908800, 0).UTC(),
				),
			),
			wantErr: false,
		},
		{
			name:   "rfc3339 window",
			window: "2020-04-01T00:00:00Z,2020-04-03T00:00:00Z",
			now:    now,
			expected: NewWindow(
				func(s string) *time.Time {
					t, _ := time.Parse(time.RFC3339, s)
					return &t
				}("2020-04-01T00:00:00Z"),
				func(s string) *time.Time {
					t, _ := time.Parse(time.RFC3339, s)
					return &t
				}("2020-04-03T00:00:00Z"),
			),
			wantErr: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseWindow(testCase.window, testCase.now)
			if (err != nil) != testCase.wantErr {
				t.Errorf("parseWindow() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}
			t.Logf("parseWindow() got start = %v, end %v", *got.Start(), *got.End())
			t.Logf("parseWindow() expected start = %v, end %v", *testCase.expected.Start(), *testCase.expected.End())
			if !got.Equal(testCase.expected) {
				t.Errorf("failed to parseWindow(%v)  \n got = from %v to %v \n expected = from %v to %v", testCase.window, *got.Start(), *got.End(), *testCase.expected.Start(), *testCase.expected.End())
			}
		})
	}
}
