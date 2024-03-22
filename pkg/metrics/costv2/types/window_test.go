package costv2

import (
	"testing"
	"time"
)

func TestParseWindow(t *testing.T) {
	now := time.Now()

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
			//window: "1d",
			now: now,
			expected: NewWindow(
				func(t time.Time) *time.Time { return &t }(now.Truncate(24*time.Hour).Add(-8*time.Hour)),
				&now,
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
