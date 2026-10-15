package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/storage"

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

// auditCtx is ctx with the actor for the audit log: the caller when there is
// one, otherwise whatever the handler attached (such as the address a sign-in
// came from).
func auditCtx(ctx context.Context) context.Context {
	if CallerFrom(ctx) != nil {
		return enrichAuditCtx(ctx)
	}
	return ctx
}

// recordAudit records a change the storage layer doesn't audit itself, such
// as a sign-in, a password change or a role grant. details is stored as JSON
// in the audit log, which more roles can read than the records themselves, so
// it holds names and IDs, never secrets or webhook URLs.
func recordAudit(ctx context.Context, store storage.AuditStorage, action, resource, resourceID string, details any) {
	store.RecordAudit(auditCtx(ctx), action, resource, resourceID, details)
}

// recordAuditFailure records a refused attempt.
func recordAuditFailure(ctx context.Context, store storage.AuditStorage, action, resource, resourceID string, details any, reason string) {
	store.RecordAuditFailure(auditCtx(ctx), action, resource, resourceID, details, reason)
}
