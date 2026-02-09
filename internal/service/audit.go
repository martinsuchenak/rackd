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
