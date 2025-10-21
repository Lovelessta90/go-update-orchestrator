package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/progress"
)

// Tracker implements in-memory progress tracking.
type Tracker struct {
	mu        sync.RWMutex
	updates   map[string]*updateState
	publisher progress.Publisher // Optional event publisher
}

// updateState holds the internal state for an update
type updateState struct {
	updateID          string
	totalDevices      int
	completedDevices  int
	failedDevices     int
	inProgressDevices int
	bytesTransferred  int64
	startTime         time.Time
	endTime           *time.Time
	deviceProgress    map[string]*progress.DeviceProgress
}

// New creates a new in-memory progress tracker.
func New() *Tracker {
	return &Tracker{
		updates: make(map[string]*updateState),
	}
}

// NewWithPublisher creates a tracker with event publishing support.
func NewWithPublisher(publisher progress.Publisher) *Tracker {
	return &Tracker{
		updates:   make(map[string]*updateState),
		publisher: publisher,
	}
}

// Start begins tracking a new update.
func (t *Tracker) Start(ctx context.Context, updateID string, totalDevices int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := &updateState{
		updateID:       updateID,
		totalDevices:   totalDevices,
		startTime:      time.Now(),
		deviceProgress: make(map[string]*progress.DeviceProgress),
	}

	t.updates[updateID] = state

	// Publish event if publisher is configured
	if t.publisher != nil {
		t.publisher.Publish(ctx, progress.Event{
			Type:     progress.EventUpdateStarted,
			UpdateID: updateID,
			Time:     time.Now(),
			Data: map[string]interface{}{
				"total_devices": totalDevices,
			},
		})
	}
}

// UpdateDevice records progress for a specific device.
func (t *Tracker) UpdateDevice(ctx context.Context, updateID, deviceID string, status string, bytesTransferred int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.updates[updateID]
	if !exists {
		return // Silently ignore updates for unknown updateID
	}

	// Get or create device progress
	deviceProg, exists := state.deviceProgress[deviceID]
	if !exists {
		deviceProg = &progress.DeviceProgress{
			DeviceID:  deviceID,
			StartTime: time.Now(),
		}
		state.deviceProgress[deviceID] = deviceProg
	}

	// Update status counters
	oldStatus := deviceProg.Status
	newStatus := core.UpdateStatus(status)

	// Decrement old status count
	switch oldStatus {
	case core.StatusInProgress:
		state.inProgressDevices--
	case core.StatusCompleted:
		state.completedDevices--
	case core.StatusFailed:
		state.failedDevices--
	}

	// Increment new status count
	switch newStatus {
	case core.StatusInProgress:
		state.inProgressDevices++
	case core.StatusCompleted:
		state.completedDevices++
		now := time.Now()
		deviceProg.EndTime = &now
	case core.StatusFailed:
		state.failedDevices++
		now := time.Now()
		deviceProg.EndTime = &now
	}

	// Update device progress
	deviceProg.Status = newStatus
	deviceProg.BytesTransferred = bytesTransferred

	// Update total bytes transferred
	state.bytesTransferred += bytesTransferred

	// Publish event if publisher is configured
	if t.publisher != nil {
		t.publisher.Publish(ctx, progress.Event{
			Type:     progress.EventDeviceUpdated,
			UpdateID: updateID,
			DeviceID: deviceID,
			Time:     time.Now(),
			Data: map[string]interface{}{
				"status":            status,
				"bytes_transferred": bytesTransferred,
			},
		})
	}
}

// Complete marks an update as completed.
func (t *Tracker) Complete(ctx context.Context, updateID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.updates[updateID]
	if !exists {
		return
	}

	now := time.Now()
	state.endTime = &now

	// Publish event if publisher is configured
	if t.publisher != nil {
		t.publisher.Publish(ctx, progress.Event{
			Type:     progress.EventUpdateCompleted,
			UpdateID: updateID,
			Time:     time.Now(),
			Data: map[string]interface{}{
				"completed_devices": state.completedDevices,
				"failed_devices":    state.failedDevices,
				"duration":          now.Sub(state.startTime),
			},
		})
	}
}

// GetProgress returns the current progress for an update.
func (t *Tracker) GetProgress(ctx context.Context, updateID string) (*progress.Progress, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.updates[updateID]
	if !exists {
		return nil, fmt.Errorf("update not found: %s", updateID)
	}

	// Calculate estimated end time
	var estimatedEnd *time.Time
	if state.completedDevices+state.failedDevices > 0 && state.inProgressDevices+state.completedDevices+state.failedDevices < state.totalDevices {
		elapsed := time.Since(state.startTime)
		completed := state.completedDevices + state.failedDevices
		avgTimePerDevice := elapsed / time.Duration(completed)
		remaining := state.totalDevices - completed
		estimatedRemaining := avgTimePerDevice * time.Duration(remaining)
		est := time.Now().Add(estimatedRemaining)
		estimatedEnd = &est
	}

	// Copy device progress map
	deviceProgressMap := make(map[string]progress.DeviceProgress, len(state.deviceProgress))
	for deviceID, dp := range state.deviceProgress {
		deviceProgressMap[deviceID] = *dp
	}

	return &progress.Progress{
		UpdateID:          state.updateID,
		TotalDevices:      state.totalDevices,
		CompletedDevices:  state.completedDevices,
		FailedDevices:     state.failedDevices,
		InProgressDevices: state.inProgressDevices,
		BytesTransferred:  state.bytesTransferred,
		StartTime:         state.startTime,
		EstimatedEnd:      estimatedEnd,
		DeviceProgress:    deviceProgressMap,
	}, nil
}
