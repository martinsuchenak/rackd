package webhook

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// EventHandler is a function that handles events
type EventHandler func(event model.Event)

// EventBus manages event publishing and subscription
type EventBus struct {
	handlers []subscription
	nextID   int
	mu       sync.RWMutex
}

type subscription struct {
	id int
	fn EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

// Subscribe calls handler for every event published, until the returned
// function is called.
func (b *EventBus) Subscribe(handler EventHandler) (unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	b.handlers = append(b.handlers, subscription{id: id, fn: handler})
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, s := range b.handlers {
			if s.id == id {
				b.handlers = append(b.handlers[:i:i], b.handlers[i+1:]...)
				return
			}
		}
	}
}

func (b *EventBus) snapshot() []EventHandler {
	b.mu.RLock()
	defer b.mu.RUnlock()
	handlers := make([]EventHandler, len(b.handlers))
	for i, s := range b.handlers {
		handlers[i] = s.fn
	}
	return handlers
}

func (b *EventBus) Publish(eventType model.EventType, payload interface{}) {
	event := model.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	handlers := b.snapshot()

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

	handlers := b.snapshot()

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
func Subscribe(handler EventHandler) (unsubscribe func()) {
	return globalEventBus.Subscribe(handler)
}
