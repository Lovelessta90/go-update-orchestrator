package events

import "context"

// Handler defines the interface for event handlers.
type Handler interface {
	// Handle processes an event.
	// Implementations should be non-blocking and handle errors internally.
	Handle(ctx context.Context, event Event)
}

// HandlerFunc is a function adapter for Handler interface.
type HandlerFunc func(ctx context.Context, event Event)

// Handle implements the Handler interface.
func (f HandlerFunc) Handle(ctx context.Context, event Event) {
	f(ctx, event)
}
