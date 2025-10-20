package orchestrator

import (
	"context"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
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

// ExecuteUpdate executes an update job across all target devices.
func (o *Orchestrator) ExecuteUpdate(ctx context.Context, update core.Update) error {
	// TODO: Implement orchestration logic:
	// 1. Validate update
	// 2. Fetch devices from registry
	// 3. Create worker pool
	// 4. Stream payload to devices via delivery mechanism
	// 5. Track progress and emit events
	// 6. Handle failures and retries
	return nil
}

// GetStatus returns the current status of an update.
func (o *Orchestrator) GetStatus(ctx context.Context, updateID string) (*core.Status, error) {
	// TODO: Query progress tracker and return status
	return nil, core.ErrUpdateNotFound
}

// Cancel attempts to cancel a running update.
func (o *Orchestrator) Cancel(ctx context.Context, updateID string) error {
	// TODO: Implement cancellation logic
	return nil
}

// Subscribe registers an event handler.
func (o *Orchestrator) Subscribe(eventType events.EventType, handler events.Handler) {
	o.events.Subscribe(eventType, handler)
}
