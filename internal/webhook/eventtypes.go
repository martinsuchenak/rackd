package webhook

import (
	"fmt"
	"sync"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// EventTypeInfo is an event type webhooks can subscribe to.
type EventTypeInfo struct {
	Type  model.EventType
	Label string
}

var eventTypes = struct {
	sync.RWMutex
	list   []EventTypeInfo
	byType map[model.EventType]bool
}{byType: map[model.EventType]bool{}}

func init() {
	for _, t := range []EventTypeInfo{
		{model.EventTypeDeviceCreated, "Device Created"},
		{model.EventTypeDeviceUpdated, "Device Updated"},
		{model.EventTypeDeviceDeleted, "Device Deleted"},
		{model.EventTypeDevicePromoted, "Device Promoted"},
		{model.EventTypeNetworkCreated, "Network Created"},
		{model.EventTypeNetworkUpdated, "Network Updated"},
		{model.EventTypeNetworkDeleted, "Network Deleted"},
		{model.EventTypeDiscoveryStarted, "Discovery Started"},
		{model.EventTypeDiscoveryCompleted, "Discovery Completed"},
		{model.EventTypeDeviceDiscovered, "Device Discovered"},
		{model.EventTypeConflictDetected, "Conflict Detected"},
		{model.EventTypeConflictResolved, "Conflict Resolved"},
		{model.EventTypePoolUtilization, "Pool Utilization High"},
	} {
		if err := RegisterEventType(t.Type, t.Label); err != nil {
			panic(err)
		}
	}
}

// RegisterEventType adds an event type that webhooks can subscribe to. Core's
// types are registered at start-up; extensions add theirs from their manifest.
func RegisterEventType(t model.EventType, label string) error {
	if t == "" {
		return fmt.Errorf("event type is empty")
	}
	eventTypes.Lock()
	defer eventTypes.Unlock()
	if label == "" {
		label = string(t)
	}
	if eventTypes.byType[t] {
		// Registering the same type and label again is a no-op, so a process
		// that builds more than one server (tests) can register a feature twice.
		for _, e := range eventTypes.list {
			if e.Type == t && e.Label == label {
				return nil
			}
		}
		return fmt.Errorf("event type %q is already registered", t)
	}
	eventTypes.byType[t] = true
	eventTypes.list = append(eventTypes.list, EventTypeInfo{Type: t, Label: label})
	return nil
}

// IsKnownEventType reports whether t is registered.
func IsKnownEventType(t model.EventType) bool {
	eventTypes.RLock()
	defer eventTypes.RUnlock()
	return eventTypes.byType[t]
}

// EventTypes returns the registered event types in registration order.
func EventTypes() []EventTypeInfo {
	eventTypes.RLock()
	defer eventTypes.RUnlock()
	return append([]EventTypeInfo(nil), eventTypes.list...)
}
