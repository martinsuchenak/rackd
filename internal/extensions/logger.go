package extensions

import (
	"context"
	"log/slog"

	"github.com/martinsuchenak/rackd/internal/log"
)

// newLogger returns a slog logger that writes through rackd's logger with the
// extension's name attached, so extension logs reach the same output and the
// in-app log viewer.
func newLogger(name string) *slog.Logger {
	return slog.New(&logHandler{attrs: []any{"extension", name}})
}

type logHandler struct {
	attrs []any
	group string
}

func (h *logHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *logHandler) Handle(_ context.Context, r slog.Record) error {
	kv := append([]any(nil), h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		kv = append(kv, h.key(a.Key), a.Value.Resolve().Any())
		return true
	})
	switch {
	case r.Level >= slog.LevelError:
		log.Error(r.Message, kv...)
	case r.Level >= slog.LevelWarn:
		log.Warn(r.Message, kv...)
	case r.Level >= slog.LevelInfo:
		log.Info(r.Message, kv...)
	default:
		log.Debug(r.Message, kv...)
	}
	return nil
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &logHandler{attrs: append([]any(nil), h.attrs...), group: h.group}
	for _, a := range attrs {
		next.attrs = append(next.attrs, h.key(a.Key), a.Value.Resolve().Any())
	}
	return next
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &logHandler{attrs: h.attrs, group: h.key(name)}
}

func (h *logHandler) key(k string) string {
	if h.group == "" {
		return k
	}
	return h.group + "." + k
}
