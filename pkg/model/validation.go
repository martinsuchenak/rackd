package model

import "fmt"

// FieldError is a validation failure on one field of a model, so callers can
// tell bad input from a storage failure with errors.As.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Message }

func fieldErr(field, format string, args ...any) error {
	return &FieldError{Field: field, Message: fmt.Sprintf(format, args...)}
}
