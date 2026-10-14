package extension

import "context"

// CallerType says who is making a request.
type CallerType int

const (
	CallerTypeAnonymous CallerType = iota
	CallerTypeUser
	CallerTypeAPIKey
	CallerTypeSystem
)

// Caller is the identity a request runs as. Rackd's authentication sets it on
// every authenticated request; permission checks and the audit log read it.
type Caller struct {
	Type      CallerType
	UserID    string
	Username  string
	IPAddress string
	Source    string
	Scopes    []string // OAuth token scopes; if non-nil, limits effective permissions
}

// IsSystem reports whether the caller is Rackd itself (a worker or startup
// task), which bypasses permission checks.
func (c *Caller) IsSystem() bool {
	return c.Type == CallerTypeSystem
}

type callerKey struct{}

// WithCaller returns a context that carries c.
func WithCaller(ctx context.Context, c *Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, c)
}

// CallerFrom returns the caller in ctx, or nil.
func CallerFrom(ctx context.Context) *Caller {
	if c, ok := ctx.Value(callerKey{}).(*Caller); ok {
		return c
	}
	return nil
}

// SystemContext returns a context that runs as Rackd itself, for background
// work. source names the work in the audit log, for example "my-feature-sync".
func SystemContext(ctx context.Context, source string) context.Context {
	return WithCaller(ctx, &Caller{Type: CallerTypeSystem, Source: source})
}
