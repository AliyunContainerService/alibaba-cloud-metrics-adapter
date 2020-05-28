package utils

import (
	"testing"
	"time"
)

func TestJudgeWithPeriod(t *testing.T) {
	t.Run("greater than period", func(t *testing.T) {
		start := time.Now().Add(-60 * time.Second)
		end := time.Now()
		err := JudgeWithPeriod(start, end, 50)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("smaller than period", func(t *testing.T) {
		start := time.Now().Add(-60 * time.Second)
		end := time.Now()
		err := JudgeWithPeriod(start, end, 90)
		if err == nil {
			t.Errorf("JudgeWithPeriod logic is not right.")
		}
	})
}
