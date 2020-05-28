package utils

import (
	"fmt"
	"strconv"
	"time"
)

func JudgeWithPeriod(startTime, endTime time.Time, period int) error {
	periodStr := strconv.Itoa(period)
	per, err := strconv.ParseInt(periodStr, 10, 64)
	if err != nil {
		return err
	}

	if (endTime.Unix() - startTime.Unix()) < per {
		return fmt.Errorf("Pls make sure that starttime minus endtime is greater than period.")
	}

	return nil
}
