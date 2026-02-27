package webhook

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

// EventHandler is a function that handles events
type EventHandler func(event model.Event)

// EventBus manages event publishing and subscription
type EventBus struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe registers a handler to receive all events
func (b *EventBus) Subscribe(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Publish sends an event to all subscribers
func (b *EventBus) Publish(eventType model.EventType, payload interface{}) {
	event := model.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	// Call handlers asynchronously
	for _, handler := range handlers {
		go handler(event)
	}
}

// PublishSync sends an event to all subscribers synchronously
func (b *EventBus) PublishSync(eventType model.EventType, payload interface{}) {
	event := model.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// ToJSON serializes an event to JSON
func ToJSON(event model.Event) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// globalEventBus is the default event bus instance
var globalEventBus = NewEventBus()

// GetEventBus returns the global event bus
func GetEventBus() *EventBus {
	return globalEventBus
}

// Publish publishes an event to the global event bus
func Publish(eventType model.EventType, payload interface{}) {
	globalEventBus.Publish(eventType, payload)
}

// Subscribe registers a handler with the global event bus
func Subscribe(handler EventHandler) {
	globalEventBus.Subscribe(handler)
}
