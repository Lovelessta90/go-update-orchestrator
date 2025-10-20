package progress

import (
	"context"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Tracker defines the interface for tracking update progress.
type Tracker interface {
	// Start begins tracking a new update.
	Start(ctx context.Context, updateID string, totalDevices int)

	// UpdateDevice records progress for a specific device.
	UpdateDevice(ctx context.Context, updateID, deviceID string, status string, bytesTransferred int64)

	// Complete marks an update as completed.
	Complete(ctx context.Context, updateID string)

	// GetProgress returns the current progress for an update.
	GetProgress(ctx context.Context, updateID string) (*Progress, error)
}

// Progress represents the current progress of an update.
type Progress struct {
	UpdateID          string
	TotalDevices      int
	CompletedDevices  int
	FailedDevices     int
	InProgressDevices int
	BytesTransferred  int64
	StartTime         time.Time
	EstimatedEnd      *time.Time
	DeviceProgress    map[string]DeviceProgress
}

// DeviceProgress represents progress for a single device.
type DeviceProgress struct {
	DeviceID         string
	Status           core.UpdateStatus
	BytesTransferred int64
	StartTime        time.Time
	EndTime          *time.Time
	Error            error
}

// EventType represents the type of progress event.
type EventType string

const (
	EventUpdateStarted   EventType = "update_started"
	EventUpdateCompleted EventType = "update_completed"
	EventDeviceUpdated   EventType = "device_updated"
)

// Event represents a progress event.
type Event struct {
	Type     EventType
	UpdateID string
	DeviceID string
	Time     time.Time
	Data     map[string]interface{}
}

// Publisher defines the interface for publishing progress events.
type Publisher interface {
	Publish(ctx context.Context, event Event)
}
