package memory

import (
	"context"
	"sync"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Registry implements an in-memory device registry.
type Registry struct {
	mu      sync.RWMutex
	devices map[string]core.Device
}

// New creates a new in-memory registry.
func New() *Registry {
	return &Registry{
		devices: make(map[string]core.Device),
	}
}

// List returns devices matching the given filter.
func (r *Registry) List(ctx context.Context, filter core.Filter) ([]core.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]core.Device, 0)

	for _, device := range r.devices {
		if matchesFilter(device, filter) {
			devices = append(devices, device)
		}
	}

	// Apply pagination
	start := filter.Offset
	end := len(devices)

	if start >= len(devices) {
		return []core.Device{}, nil
	}

	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}

	return devices[start:end], nil
}

// matchesFilter checks if a device matches the given filter criteria.
func matchesFilter(device core.Device, filter core.Filter) bool {
	// Filter by specific IDs
	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if device.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by status
	if filter.Status != nil && device.Status != *filter.Status {
		return false
	}

	// Filter by location
	if filter.Location != "" && device.Location != filter.Location {
		return false
	}

	// Filter by metadata tags
	for key, value := range filter.Tags {
		if deviceValue, ok := device.Metadata[key]; !ok || deviceValue != value {
			return false
		}
	}

	// Filter by last seen time
	if filter.LastSeenBefore != nil && device.LastSeen != nil {
		if device.LastSeen.After(*filter.LastSeenBefore) {
			return false
		}
	}

	if filter.LastSeenAfter != nil && device.LastSeen != nil {
		if device.LastSeen.Before(*filter.LastSeenAfter) {
			return false
		}
	}

	// Note: Firmware version filtering (MinFirmware, MaxFirmware) would require
	// semantic version comparison, which we'll skip for now

	return true
}

// Get retrieves a single device by ID.
func (r *Registry) Get(ctx context.Context, id string) (*core.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, ok := r.devices[id]
	if !ok {
		return nil, core.ErrDeviceNotFound
	}
	return &device, nil
}

// Add registers a new device.
func (r *Registry) Add(ctx context.Context, device core.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices[device.ID] = device
	return nil
}

// Update modifies an existing device.
func (r *Registry) Update(ctx context.Context, device core.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.devices[device.ID]; !ok {
		return core.ErrDeviceNotFound
	}
	r.devices[device.ID] = device
	return nil
}

// Delete removes a device from the registry.
func (r *Registry) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.devices[id]; !ok {
		return core.ErrDeviceNotFound
	}
	delete(r.devices, id)
	return nil
}
