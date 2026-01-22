package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const (
	ExitSuccess      = 0
	ExitGeneric      = 1
	ExitInvalidUsage = 2
	ExitNetwork      = 3
	ExitAuth         = 4
	ExitServer       = 5
)

type APIError struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

func HandleError(resp *http.Response) error {
	var errResp APIError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("%s: %s", errResp.Code, errResp.Error)
}

func PrintError(msg string, code int) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(code)
}
