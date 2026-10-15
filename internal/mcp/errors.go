package mcp

import (
	"errors"

	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/mcp"
)

// Application error codes, in the JSON-RPC implementation-defined range.
const (
	errCodeUnauthenticated = -32001
	errCodeForbidden       = -32003
	errCodeNotFound        = -32004
	errCodeConflict        = -32009
	errCodeUpstream        = -32010 // a DNS provider or other outside service failed
)

// storageNotFound are the not-found errors services can pass through from
// storage unchanged.
var storageNotFound = []error{
	storage.ErrAPIKeyNotFound, storage.ErrCircuitNotFound, storage.ErrConflictNotFound,
	storage.ErrCustomFieldNotFound, storage.ErrDNSProviderNotFound, storage.ErrDNSRecordNotFound,
	storage.ErrDNSZoneNotFound, storage.ErrDatacenterNotFound, storage.ErrDeliveryNotFound,
	storage.ErrDeviceNotFound, storage.ErrDiscoveryNotFound, storage.ErrNATNotFound,
	storage.ErrNetworkNotFound, storage.ErrPoolNotFound, storage.ErrReservationNotFound,
	storage.ErrRoleNotFound, storage.ErrScanNotFound, storage.ErrUserNotFound, storage.ErrWebhookNotFound,
	storage.ErrProfileNotFound, storage.ErrScheduledScanNotFound,
}

// toolError turns a service error into an MCP error a client can act on:
// invalid parameters, forbidden, not found or conflict, with the service's
// message. Anything else is logged and reported only as an internal error, so
// database and other internal details don't reach the client, as on the REST
// API.
func toolError(err error) error {
	switch {
	case errors.Is(err, service.ErrValidation):
		return mcp.NewToolErrorInvalidParams(err.Error())
	case errors.As(err, new(*service.PrivilegeError)):
		return mcp.NewToolError(errCodeForbidden, err.Error(), nil)
	case errors.Is(err, service.ErrForbidden):
		return mcp.NewToolError(errCodeForbidden, "Forbidden", nil)
	case errors.Is(err, service.ErrUnauthenticated):
		return mcp.NewToolError(errCodeUnauthenticated, "Unauthorized", nil)
	case errors.Is(err, service.ErrDNSProviderTestFailed):
		// Like REST's 502: the cause is the outside service's, not Rackd's.
		return mcp.NewToolError(errCodeUpstream, err.Error(), nil)
	case errors.Is(err, service.ErrNotFound), isStorageNotFound(err), errors.Is(err, discovery.ErrScanNotFound):
		return mcp.NewToolError(errCodeNotFound, err.Error(), nil)
	case errors.Is(err, service.ErrAlreadyExists), errors.Is(err, service.ErrLastAdmin),
		errors.Is(err, service.ErrIPNotAvailable), errors.Is(err, discovery.ErrTooManyScans),
		errors.Is(err, discovery.ErrScanNotRunning):
		return mcp.NewToolError(errCodeConflict, err.Error(), nil)
	case errors.Is(err, service.ErrSystemRole), errors.Is(err, service.ErrSelfDelete):
		return mcp.NewToolErrorInvalidParams(err.Error())
	}
	var toolErr *mcp.ToolError
	if errors.As(err, &toolErr) {
		return err
	}
	log.Error("MCP tool failed", "error", err)
	return mcp.NewToolErrorInternal("Internal error")
}

func isStorageNotFound(err error) bool {
	for _, target := range storageNotFound {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
