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
	// Owner is the extension that declared the type, or empty for core's.
	Owner string
}

var eventTypes = struct {
	sync.RWMutex
	list   []EventTypeInfo
	byType map[model.EventType]bool
}{byType: map[model.EventType]bool{}}

func init() {
	for _, t := range []struct {
		Type  model.EventType
		Label string
	}{
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

// RegisterEventType adds one of core's event types.
func RegisterEventType(t model.EventType, label string) error {
	return RegisterExtensionEventType("", t, label)
}

// RegisterExtensionEventType adds an event type declared by the named
// extension. A type can't be claimed by two owners, so an extension can't
// declare (and then publish) one of core's types. Registering the same type,
// label and owner again is a no-op, so a process that builds more than one
// server (tests) can register a feature twice.
func RegisterExtensionEventType(owner string, t model.EventType, label string) error {
	if t == "" {
		return fmt.Errorf("event type is empty")
	}
	if label == "" {
		label = string(t)
	}
	eventTypes.Lock()
	defer eventTypes.Unlock()
	if eventTypes.byType[t] {
		for _, e := range eventTypes.list {
			if e.Type == t {
				if e.Owner == owner && e.Label == label {
					return nil
				}
				by := e.Owner
				if by == "" {
					by = "core"
				}
				return fmt.Errorf("event type %q is already registered by %s", t, by)
			}
		}
	}
	eventTypes.byType[t] = true
	eventTypes.list = append(eventTypes.list, EventTypeInfo{Type: t, Label: label, Owner: owner})
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
