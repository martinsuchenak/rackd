package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// APIError is the API's error body. details is a list of field errors for
// validation failures, and may be other JSON elsewhere.
type APIError struct {
	Error   string          `json:"error"`
	Code    string          `json:"code"`
	Details json.RawMessage `json:"details,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// HandleError turns an error response into an error that says what went
// wrong, including each field a validation error names.
func HandleError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var errResp APIError
	if err := json.Unmarshal(body, &errResp); err != nil || (errResp.Error == "" && errResp.Code == "") {
		if text := strings.TrimSpace(string(body)); text != "" && len(text) < 300 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, text)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	msg := errResp.Error
	if errResp.Code != "" {
		msg = errResp.Code + ": " + msg
	}
	var fields []fieldError
	if json.Unmarshal(errResp.Details, &fields) == nil && len(fields) > 0 {
		parts := make([]string, 0, len(fields))
		for _, f := range fields {
			if f.Field != "" {
				parts = append(parts, f.Field+": "+f.Message)
			} else {
				parts = append(parts, f.Message)
			}
		}
		msg += " (" + strings.Join(parts, "; ") + ")"
	}
	return errors.New(msg)
}
