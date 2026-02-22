package task

import "errors"

var (
	ErrTaskNotFound = errors.New("task not found")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return "invalid " + e.Field + ": " + e.Message
}
