package costv2

import (
	"testing"
	"time"
)

func TestIsValidDurationString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1d", true},
		{"1h", true},
		{"10m", true},
		{"2h", true},
		{"3w", true},
		{"1s", true},
		{"", false},
		{"1y", false},
		{"1h30m", false},
		{"100ms", false},
		{"-1h", false},
		{"5h4m", false},
		{"1D", false},
		{"10S", false},
		{"h", false},
	}

	for _, test := range tests {
		if got, _ := IsValidDurationString(test.input); got != test.expected {
			t.Errorf("isValidPromQLDurationString(%q) = %v; want %v", test.input, got, test.expected)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1d", time.Duration(24 * time.Hour)},
		{"1h", time.Duration(1 * time.Hour)},
		{"10m", time.Duration(10 * time.Minute)},
		{"2h", time.Duration(2 * time.Hour)},
		{"3w", time.Duration(3 * 7 * 24 * time.Hour)},
		{"1s", time.Duration(1 * time.Second)},
	}

	for _, test := range tests {
		if got, _ := ParseDuration(test.input); got != test.expected {
			t.Errorf("ParseDuration(%q) = %v; want %v", test.input, got, test.expected)
		}
	}
}
