package audit

import (
	"context"
)

type contextKey struct{}

// Context contains audit information for a request
type Context struct {
	UserID    string // API key ID or user ID
	Username  string // API key name or username
	IPAddress string // Client IP address (for API requests)
	Source    string // Entry point: api, mcp, cli, discovery, scheduler
}

// WithContext wraps a context with audit information
func WithContext(parentCtx context.Context, auditCtx *Context) context.Context {
	if auditCtx == nil {
		return parentCtx
	}
	return context.WithValue(parentCtx, contextKey{}, auditCtx)
}

// FromContext extracts audit information from a context
// Returns (context, true) if found, (nil, false) otherwise
func FromContext(ctx context.Context) (*Context, bool) {
	if ctx == nil {
		return nil, false
	}
	auditCtx, ok := ctx.Value(contextKey{}).(*Context)
	if !ok {
		return nil, false
	}
	return auditCtx, true
}

// MustFromContext extracts audit information from a context
// Panics if audit context is not found
func MustFromContext(ctx context.Context) *Context {
	auditCtx, ok := FromContext(ctx)
	if !ok {
		panic("audit context not found in context")
	}
	return auditCtx
}
