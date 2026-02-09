package service

import (
	"context"
)

type CallerType int

const (
	CallerTypeAnonymous CallerType = iota
	CallerTypeUser
	CallerTypeAPIKey
	CallerTypeSystem
)

type Caller struct {
	Type      CallerType
	UserID    string
	Username  string
	IPAddress string
	Source    string
}

func (c *Caller) IsSystem() bool {
	return c.Type == CallerTypeSystem
}

type contextKey string

const callerKey contextKey = "caller"

func WithCaller(ctx context.Context, c *Caller) context.Context {
	return context.WithValue(ctx, callerKey, c)
}

func CallerFrom(ctx context.Context) *Caller {
	if c, ok := ctx.Value(callerKey).(*Caller); ok {
		return c
	}
	return nil
}

func SystemContext(ctx context.Context, source string) context.Context {
	return WithCaller(ctx, &Caller{
		Type:   CallerTypeSystem,
		Source: source,
	})
}
