package extension

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Errors returned by Host and expected from feature code. WriteError maps them
// to HTTP statuses; wrap them with fmt.Errorf("...: %w", ErrNotFound) to add
// detail.
var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrValidation      = errors.New("validation error")
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthenticated = errors.New("unauthenticated")
)

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes err as Rackd's standard JSON error response, choosing the
// status from the errors above. Any other error is a 500 whose message is not
// shown to the client.
func WriteError(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", err.Error()
	case errors.Is(err, ErrAlreadyExists):
		status, code, message = http.StatusConflict, "ALREADY_EXISTS", err.Error()
	case errors.Is(err, ErrValidation):
		status, code, message = http.StatusBadRequest, "VALIDATION_ERROR", err.Error()
	case errors.Is(err, ErrForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "Forbidden"
	case errors.Is(err, ErrUnauthenticated):
		status, code, message = http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized"
	}
	WriteJSON(w, status, map[string]string{"error": message, "code": code})
}

// DecodeJSON decodes the request body into v, returning an ErrValidation error
// for malformed JSON. The body is already limited to 1 MiB.
func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return errors.Join(ErrValidation, err)
	}
	return nil
}
