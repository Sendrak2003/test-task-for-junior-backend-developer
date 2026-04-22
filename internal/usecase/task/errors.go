package task

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid task input")
	ErrInvalidRecurrence = errors.New("invalid recurrence")
)
