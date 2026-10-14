package extensions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// host implements extension.Host for one feature.
type host struct {
	name   string
	rt     *Runtime
	logger *slog.Logger
}

func (h *host) Core() extension.Core { return core{h.rt} }

func (h *host) Events() extension.EventBus { return events{h} }

func (h *host) Authorize(ctx context.Context, resource, action string) error {
	return h.rt.svc.Authorize(ctx, resource, action)
}

func (h *host) Audit(ctx context.Context, action, resource, resourceID string, details any) {
	h.rt.store.RecordAudit(service.WithAuditContext(ctx), action, resource, resourceID, details)
}

func (h *host) DB() xdal.DBInterface { return noClose{h.rt.store.XDAL()} }

func (h *host) Logger() *slog.Logger { return h.logger }

func (h *host) Go(name string, fn func(ctx context.Context)) {
	h.rt.wg.Add(1)
	go func() {
		defer h.rt.wg.Done()
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("Extension background work panicked", "extension", h.name, "work", name, "panic", fmt.Sprint(rec), "stack", string(debug.Stack()))
			}
		}()
		fn(h.rt.ctx)
	}()
}

// core gives features core's services, which check the caller's permissions.
type core struct{ rt *Runtime }

func (c core) Devices() extension.Devices   { return c.rt.svc.Devices }
func (c core) Networks() extension.Networks { return c.rt.svc.Networks }

// events publishes on the webhook event bus, which webhooks deliver from.
type events struct{ h *host }

func (e events) Publish(eventType string, payload any) {
	t := model.EventType(eventType)
	if !webhook.IsKnownEventType(t) {
		e.h.logger.Error("Event type is not registered; declare it in the manifest", "event_type", eventType)
		return
	}
	webhook.Publish(t, payload)
}

func (e events) Subscribe(fn func(model.Event)) { webhook.Subscribe(fn) }

// noClose stops a feature from closing core's database connection.
type noClose struct{ xdal.DBInterface }

func (noClose) Close() error {
	return errors.New("extension: the database is owned by rackd and cannot be closed")
}
