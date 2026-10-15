package mcp

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/paularlott/mcp"
)

func TestToolErrorKeepsUsefulMessagesAndHidesInternalOnes(t *testing.T) {
	for _, tc := range []struct {
		err      error
		code     int
		contains string
	}{
		{service.ValidationErrors{{Field: "name", Message: "Name is required"}}, mcp.ErrorCodeInvalidParams, "Name is required"},
		{service.ErrForbidden, errCodeForbidden, "Forbidden"},
		{fmt.Errorf("get: %w", storage.ErrNetworkNotFound), errCodeNotFound, "network not found"},
		{fmt.Errorf("%w: connection refused", service.ErrDNSProviderTestFailed), errCodeUpstream, "connection refused"},
		{errors.New("UNIQUE constraint failed: networks.name"), errCodeConflict, "name is already in use"},
		{&model.FieldError{Field: "type", Message: "invalid credential type"}, mcp.ErrorCodeInvalidParams, "invalid credential type"},
		{fmt.Errorf("%w: timeout", service.ErrDNSProviderFailed), errCodeUpstream, "timeout"},
		{errors.New("database is locked: SQLITE_BUSY"), mcp.ErrorCodeInternalError, "Internal error"},
	} {
		var toolErr *mcp.ToolError
		if !errors.As(toolError(tc.err), &toolErr) {
			t.Fatalf("%v: not a ToolError", tc.err)
		}
		if toolErr.Code != tc.code || toolErr.Message == "" || !strings.Contains(toolErr.Message, tc.contains) {
			t.Errorf("%v -> %d %q, want %d containing %q", tc.err, toolErr.Code, toolErr.Message, tc.code, tc.contains)
		}
		if tc.code == mcp.ErrorCodeInternalError && strings.Contains(toolErr.Message, "SQLITE") {
			t.Errorf("internal detail reached the client: %q", toolErr.Message)
		}
	}
}
