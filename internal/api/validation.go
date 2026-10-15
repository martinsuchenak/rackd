package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// uuidPattern is the canonical UUID form Rackd assigns.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateNewID checks the id a client sends when creating something. Rackd
// assigns an id when it is empty; a client may supply one, for example when
// importing, but only a UUID. IDs end up in URLs and exports, so a chosen
// string like "../users/<id>" must never be stored.
func ValidateNewID(id string) ValidationErrors {
	if id == "" || uuidPattern.MatchString(id) {
		return nil
	}
	return ValidationErrors{{Field: "id", Message: "id must be a UUID, or omitted so Rackd assigns one"}}
}

// timeRangeParams reads the start_time and end_time query parameters (RFC
// 3339). A value that doesn't parse is an error rather than ignored, which
// would silently widen the filter to everything.
func timeRangeParams(r *http.Request) (start, end *time.Time, errs ValidationErrors) {
	parse := func(name string) *time.Time {
		v := r.URL.Query().Get(name)
		if v == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			errs = append(errs, ValidationError{Field: name, Message: name + " must be an RFC 3339 time, such as 2026-10-15T00:00:00Z"})
			return nil
		}
		return &t
	}
	start, end = parse("start_time"), parse("end_time")
	if start != nil && end != nil && end.Before(*start) {
		errs = append(errs, ValidationError{Field: "end_time", Message: "end_time is before start_time"})
	}
	return start, end, errs
}
