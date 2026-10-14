package service

import (
	"context"
	"slices"

	wh "github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// publishEvent announces a stored change to subscribers: webhooks and
// extensions. Delivery is asynchronous, so payloads are copies that later
// changes to the caller's record can't alter.
var publishEvent = func(t model.EventType, payload any) { wh.Publish(t, payload) }

func deviceSnapshot(d *model.Device) *model.Device {
	c := *d
	c.Tags = slices.Clone(d.Tags)
	c.Addresses = slices.Clone(d.Addresses)
	c.Domains = slices.Clone(d.Domains)
	c.CustomFields = slices.Clone(d.CustomFields)
	return &c
}

func networkSnapshot(n *model.Network) *model.Network {
	c := *n
	return &c
}

// deletedDevice returns the device as stored, for the event published after
// it is deleted, or just its ID if it can't be read.
func (s *DeviceService) deletedDevice(ctx context.Context, id string) *model.Device {
	if d, err := s.store.GetDevice(ctx, id); err == nil && d != nil {
		return d
	}
	return &model.Device{ID: id}
}
