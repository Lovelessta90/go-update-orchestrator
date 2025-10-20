package orchestrator

import (
	"context"
	"fmt"
	"io"

	"github.com/dovaclean/go-update-orchestrator/internal/pool"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/delivery"
	"github.com/dovaclean/go-update-orchestrator/pkg/events"
	"github.com/dovaclean/go-update-orchestrator/pkg/progress"
	"github.com/dovaclean/go-update-orchestrator/pkg/progress/memory"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry"
)

// NewWithTracker creates an orchestrator with a custom progress tracker.
func NewWithTracker(
	config *Config,
	reg registry.Registry,
	del delivery.Delivery,
	tracker progress.Tracker,
) (*Orchestrator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Orchestrator{
		config:   config,
		registry: reg,
		delivery: del,
		events:   events.NewBus(config.EventBufferSize),
		progress: tracker,
	}, nil
}

// NewDefault creates an orchestrator with default progress tracker.
func NewDefault(
	config *Config,
	reg registry.Registry,
	del delivery.Delivery,
) (*Orchestrator, error) {
	return NewWithTracker(config, reg, del, memory.New())
}

// ExecuteUpdateWithPayload executes an update with a provided payload reader.
func (o *Orchestrator) ExecuteUpdateWithPayload(ctx context.Context, update core.Update, payload io.ReadSeeker) error {
	// 1. Validate update
	if update.ID == "" {
		return fmt.Errorf("update ID is required")
	}

	// 2. Fetch devices from registry
	devices, err := o.registry.List(ctx, *update.DeviceFilter)
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no devices match the filter")
	}

	// 3. Start progress tracking
	o.progress.Start(ctx, update.ID, len(devices))

	// Emit update started event
	o.events.Publish(ctx, events.Event{
		Type:      events.EventUpdateStarted,
		UpdateID:  update.ID,
		DeviceID:  "",
		Timestamp: update.CreatedAt,
		Data: map[string]interface{}{
			"total_devices": len(devices),
			"strategy":      update.Strategy,
		},
	})

	// 4. Create worker pool
	workerPool := pool.New(o.config.MaxConcurrent)
	workerPool.Start(ctx)
	defer workerPool.Stop()

	// 5. Submit device update tasks
	for _, device := range devices {
		device := device // Capture for closure
		workerPool.Submit(func(ctx context.Context) error {
			return o.updateDevice(ctx, update, device, payload)
		})
	}

	// 6. Wait for all tasks to complete (handled by pool.Stop())
	// Note: Stop() is called via defer above

	// 7. Mark update as complete
	o.progress.Complete(ctx, update.ID)

	// Emit update completed event
	prog, _ := o.progress.GetProgress(ctx, update.ID)
	o.events.Publish(ctx, events.Event{
		Type:      events.EventUpdateCompleted,
		UpdateID:  update.ID,
		DeviceID:  "",
		Timestamp: update.CreatedAt,
		Data: map[string]interface{}{
			"completed": prog.CompletedDevices,
			"failed":    prog.FailedDevices,
		},
	})

	return nil
}

// updateDevice handles the update for a single device.
func (o *Orchestrator) updateDevice(ctx context.Context, update core.Update, device core.Device, payload io.ReadSeeker) error {
	// Mark device as in progress
	o.progress.UpdateDevice(ctx, update.ID, device.ID, string(core.StatusInProgress), 0)

	// Emit device started event
	o.events.Publish(ctx, events.Event{
		Type:      events.EventDeviceStarted,
		UpdateID:  update.ID,
		DeviceID:  device.ID,
		Timestamp: update.CreatedAt,
		Data: map[string]interface{}{
			"device_address": device.Address,
		},
	})

	// Reset payload to beginning for this device
	if _, err := payload.Seek(0, io.SeekStart); err != nil {
		o.handleDeviceFailure(ctx, update, device, err)
		return err
	}

	// Push update to device
	err := o.delivery.Push(ctx, device, payload)

	if err != nil {
		o.handleDeviceFailure(ctx, update, device, err)
		return err
	}

	// Mark device as completed
	o.progress.UpdateDevice(ctx, update.ID, device.ID, string(core.StatusCompleted), 0)

	// Emit device completed event
	o.events.Publish(ctx, events.Event{
		Type:      events.EventDeviceCompleted,
		UpdateID:  update.ID,
		DeviceID:  device.ID,
		Timestamp: update.CreatedAt,
		Data: map[string]interface{}{
			"success": true,
		},
	})

	return nil
}

// handleDeviceFailure handles a failed device update.
func (o *Orchestrator) handleDeviceFailure(ctx context.Context, update core.Update, device core.Device, err error) {
	// Mark device as failed
	o.progress.UpdateDevice(ctx, update.ID, device.ID, string(core.StatusFailed), 0)

	// Emit device failed event
	o.events.Publish(ctx, events.Event{
		Type:      events.EventDeviceFailed,
		UpdateID:  update.ID,
		DeviceID:  device.ID,
		Timestamp: update.CreatedAt,
		Data: map[string]interface{}{
			"error": err.Error(),
		},
		Error: err,
	})
}

// GetStatus returns the current status of an update.
func (o *Orchestrator) GetStatus(ctx context.Context, updateID string) (*core.Status, error) {
	prog, err := o.progress.GetProgress(ctx, updateID)
	if err != nil {
		return nil, err
	}

	// Convert progress to status
	status := &core.Status{
		UpdateID:     prog.UpdateID,
		TotalDevices: prog.TotalDevices,
		Completed:    prog.CompletedDevices,
		Failed:       prog.FailedDevices,
		InProgress:   prog.InProgressDevices,
		StartedAt:    prog.StartTime,
		EstimatedEnd: prog.EstimatedEnd,
		DeviceStatus: make(map[string]string),
	}

	// Convert device progress to device status map
	for deviceID, deviceProg := range prog.DeviceProgress {
		status.DeviceStatus[deviceID] = string(deviceProg.Status)
	}

	// Determine overall status
	if prog.CompletedDevices+prog.FailedDevices == prog.TotalDevices {
		if prog.FailedDevices > 0 {
			status.Status = core.StatusFailed
		} else {
			status.Status = core.StatusCompleted
		}
	} else {
		status.Status = core.StatusInProgress
	}

	return status, nil
}
