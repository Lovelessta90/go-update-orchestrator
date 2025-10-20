package registry

import (
	"context"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Registry defines the interface for device storage and retrieval.
// Implementations can use in-memory, database, or external systems.
type Registry interface {
	// List returns devices matching the given filter.
	List(ctx context.Context, filter core.Filter) ([]core.Device, error)

	// Get retrieves a single device by ID.
	Get(ctx context.Context, id string) (*core.Device, error)

	// Add registers a new device.
	Add(ctx context.Context, device core.Device) error

	// Update modifies an existing device.
	Update(ctx context.Context, device core.Device) error

	// Delete removes a device from the registry.
	Delete(ctx context.Context, id string) error
}
