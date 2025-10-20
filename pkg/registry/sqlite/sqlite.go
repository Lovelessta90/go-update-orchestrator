package sqlite

import (
	"context"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Registry implements a SQLite-based device registry.
type Registry struct {
	// TODO: Add database/sql connection
}

// New creates a new SQLite registry.
func New(dbPath string) (*Registry, error) {
	// TODO: Initialize SQLite database
	return &Registry{}, nil
}

// List returns devices matching the given filter.
func (r *Registry) List(ctx context.Context, filter core.Filter) ([]core.Device, error) {
	// TODO: Implement with SQL query
	return nil, nil
}

// Get retrieves a single device by ID.
func (r *Registry) Get(ctx context.Context, id string) (*core.Device, error) {
	// TODO: Implement with SQL query
	return nil, core.ErrDeviceNotFound
}

// Add registers a new device.
func (r *Registry) Add(ctx context.Context, device core.Device) error {
	// TODO: Implement with INSERT
	return nil
}

// Update modifies an existing device.
func (r *Registry) Update(ctx context.Context, device core.Device) error {
	// TODO: Implement with UPDATE
	return nil
}

// Delete removes a device from the registry.
func (r *Registry) Delete(ctx context.Context, id string) error {
	// TODO: Implement with DELETE
	return nil
}
