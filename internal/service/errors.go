package service

import (
	"errors"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/storage"

	"github.com/martinsuchenak/rackd/pkg/extension"
)

// The common errors are shared with pkg/extension, so errors from core
// services keep their meaning when they reach extension code.
var (
	ErrNotFound        = extension.ErrNotFound
	ErrAlreadyExists   = extension.ErrAlreadyExists
	ErrValidation      = extension.ErrValidation
	ErrForbidden       = extension.ErrForbidden
	ErrUnauthenticated = extension.ErrUnauthenticated
)

var (
	ErrSystemRole     = errors.New("cannot modify system role")
	ErrSelfDelete     = errors.New("cannot delete own account")
	ErrIPNotAvailable = errors.New("no IP addresses available")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("validation error: %s", msgs)
}

// Unwrap returns ErrValidation so errors.Is(err, ErrValidation) works.
func (e ValidationErrors) Unwrap() error {
	return ErrValidation
}

// fromStoreError turns a database constraint failure into the error a client
// can act on: a duplicate becomes ErrAlreadyExists, a missing required field
// a validation error. Anything else is returned unchanged, so its text stays
// out of responses.
func fromStoreError(err error) error {
	if field, ok := storage.DuplicateField(err); ok {
		return fmt.Errorf("%w: %s is already in use", ErrAlreadyExists, field)
	}
	if field, ok := storage.MissingField(err); ok {
		return ValidationErrors{{Field: field, Message: field + " is required"}}
	}
	return err
}
