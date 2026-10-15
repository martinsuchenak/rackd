package service

import (
	"errors"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/storage"

	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
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

// storageNotFound are the not-found errors storage returns, which services
// often pass through unchanged.
var storageNotFound = []error{
	storage.ErrAPIKeyNotFound, storage.ErrCircuitNotFound, storage.ErrConflictNotFound,
	storage.ErrCustomFieldNotFound, storage.ErrDNSProviderNotFound, storage.ErrDNSRecordNotFound,
	storage.ErrDNSZoneNotFound, storage.ErrDatacenterNotFound, storage.ErrDeliveryNotFound,
	storage.ErrDeviceNotFound, storage.ErrDiscoveryNotFound, storage.ErrNATNotFound,
	storage.ErrNetworkNotFound, storage.ErrPoolNotFound, storage.ErrReservationNotFound,
	storage.ErrRoleNotFound, storage.ErrScanNotFound, storage.ErrUserNotFound, storage.ErrWebhookNotFound,
	storage.ErrProfileNotFound, storage.ErrScheduledScanNotFound, storage.ErrRuleNotFound,
	storage.ErrPermissionNotFound, storage.ErrAuditLogNotFound,
}

// notFound keeps a storage error's message while matching ErrNotFound.
type notFound struct{ error }

func (e notFound) Is(target error) bool { return target == ErrNotFound }
func (e notFound) Unwrap() error        { return e.error }

// NormalizeError turns the errors storage and the models return into the
// service errors a client can act on, so the REST API and MCP answer alike:
// a model's field error or a missing id becomes a validation error, a storage
// not-found ErrNotFound, a duplicate ErrAlreadyExists, and a missing required
// field or a broken reference a validation error. Anything else is returned
// unchanged, so its text stays out of responses.
func NormalizeError(err error) error {
	if err == nil || errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
		return err
	}
	var fieldErr *model.FieldError
	if errors.As(err, &fieldErr) {
		return ValidationErrors{{Field: fieldErr.Field, Message: fieldErr.Message}}
	}
	if errors.Is(err, storage.ErrInvalidID) {
		return ValidationErrors{{Field: "id", Message: "an id is required"}}
	}
	for _, target := range storageNotFound {
		if errors.Is(err, target) {
			return notFound{err}
		}
	}
	if field, ok := storage.DuplicateField(err); ok {
		return fmt.Errorf("%w: %s is already in use", ErrAlreadyExists, field)
	}
	if field, ok := storage.MissingField(err); ok {
		return ValidationErrors{{Field: field, Message: field + " is required"}}
	}
	if storage.IsForeignKeyFailure(err) {
		return ValidationErrors{{Field: "", Message: "refers to a record that doesn't exist, or the record is still in use"}}
	}
	return err
}
