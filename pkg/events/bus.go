package events

import (
	"context"
	"sync"
)

// Bus manages event publishing and subscription.
type Bus struct {
	mu         sync.RWMutex
	handlers   map[EventType][]Handler
	bufferSize int
}

// NewBus creates a new event bus.
func NewBus(bufferSize int) *Bus {
	return &Bus{
		handlers:   make(map[EventType][]Handler),
		bufferSize: bufferSize,
	}
}

// Subscribe registers a handler for a specific event type.
func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish sends an event to all registered handlers.
func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	// Call handlers concurrently but don't block
	for _, handler := range handlers {
		go handler.Handle(ctx, event)
	}
}

// Unsubscribe removes a handler (TODO: needs handler identification).
func (b *Bus) Unsubscribe(eventType EventType, handler Handler) {
	// TODO: Implement handler removal
}
