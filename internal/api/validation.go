package api

import (
	"fmt"
	"regexp"
	"strings"
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
