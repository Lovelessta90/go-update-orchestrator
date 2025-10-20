package core

import "context"

// Scheduler manages when and how updates are executed.
// Implementations control update timing, batching, and scheduling strategies.
type Scheduler interface {
	// Schedule queues an update for execution.
	Schedule(ctx context.Context, update Update) error

	// Status returns the current status of an update.
	Status(ctx context.Context, updateID string) (*Status, error)

	// Cancel attempts to cancel a running update.
	Cancel(ctx context.Context, updateID string) error

	// List returns all updates matching the given status.
	List(ctx context.Context, status UpdateStatus) ([]Status, error)
}
