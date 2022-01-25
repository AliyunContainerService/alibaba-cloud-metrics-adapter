package utils

import (
	"fmt"
)

const (
	DEFAULT_TIME_FORMAT = "2006-01-02 15:04:05"
	PROM_FORMAT         = "2006-01-02T15:04:05.781Z"
)

// ErrorType is the type of the API error.
type ErrorType string

const (
	ErrBadData     ErrorType = "bad_data"
	ErrTimeout               = "timeout"
	ErrCanceled              = "canceled"
	ErrExec                  = "execution"
	ErrBadResponse           = "bad_response"
)

// Error is an error returned by the API.
type Error struct {
	Type ErrorType
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Msg)
}

// ResponseStatus is the type of response from the API: succeeded or error.
type ResponseStatus string

const (
	ResponseSucceeded ResponseStatus = "succeeded"
	ResponseError                    = "error"
)

