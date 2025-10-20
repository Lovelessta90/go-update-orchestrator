package orchestrator

import (
	"context"

	"github.com/dovaclean/go-update-orchestrator/pkg/delivery"
	"github.com/dovaclean/go-update-orchestrator/pkg/events"
	"github.com/dovaclean/go-update-orchestrator/pkg/progress"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry"
)

// Orchestrator coordinates all components to execute updates.
type Orchestrator struct {
	config   *Config
	registry registry.Registry
	delivery delivery.Delivery
	events   *events.Bus
	progress progress.Tracker
	// TODO: Add scheduler when implemented
}

// New creates a new orchestrator with the given configuration and components.
func New(
	config *Config,
	registry registry.Registry,
	delivery delivery.Delivery,
) (*Orchestrator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Orchestrator{
		config:   config,
		registry: registry,
		delivery: delivery,
		events:   events.NewBus(config.EventBufferSize),
		// TODO: Initialize progress tracker
	}, nil
}

// Subscribe registers an event handler.
func (o *Orchestrator) Subscribe(eventType events.EventType, handler events.Handler) {
	o.events.Subscribe(eventType, handler)
}

// Cancel attempts to cancel a running update.
// TODO: Implement cancellation logic
func (o *Orchestrator) Cancel(ctx context.Context, updateID string) error {
	return nil
}
