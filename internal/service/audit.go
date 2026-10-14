package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/audit"
)

func enrichAuditCtx(ctx context.Context) context.Context {
	caller := CallerFrom(ctx)
	if caller == nil {
		return ctx
	}
	return audit.WithContext(ctx, &audit.Context{
		UserID:    caller.UserID,
		Username:  caller.Username,
		IPAddress: caller.IPAddress,
		Source:    caller.Source,
	})
}

// WithAuditContext returns ctx with the caller's identity attached for the
// storage-level audit log, as services do before writing.
func WithAuditContext(ctx context.Context) context.Context {
	return enrichAuditCtx(ctx)
}
