package service

import (
	"context"

	"github.com/martinsuchenak/rackd/pkg/extension"
)

// The caller types live in pkg/extension so that core and extensions share one
// context key.
type (
	CallerType = extension.CallerType
	Caller     = extension.Caller
)

const (
	CallerTypeAnonymous = extension.CallerTypeAnonymous
	CallerTypeUser      = extension.CallerTypeUser
	CallerTypeAPIKey    = extension.CallerTypeAPIKey
	CallerTypeSystem    = extension.CallerTypeSystem
)

func WithCaller(ctx context.Context, c *Caller) context.Context {
	return extension.WithCaller(ctx, c)
}

func CallerFrom(ctx context.Context) *Caller {
	return extension.CallerFrom(ctx)
}

func SystemContext(ctx context.Context, source string) context.Context {
	return extension.SystemContext(ctx, source)
}
