package events

import "time"

// EventType represents the type of event.
type EventType string

const (
	EventUpdateStarted   EventType = "update.started"
	EventUpdateCompleted EventType = "update.completed"
	EventUpdateFailed    EventType = "update.failed"
	EventUpdateCancelled EventType = "update.cancelled"

	EventDeviceStarted   EventType = "device.started"
	EventDeviceCompleted EventType = "device.completed"
	EventDeviceFailed    EventType = "device.failed"

	EventProgressUpdate EventType = "progress.update"
)

// Event represents an event in the system.
type Event struct {
	Type      EventType              // Type of event
	UpdateID  string                 // Update identifier
	DeviceID  string                 // Device identifier (if applicable)
	Timestamp time.Time              // When the event occurred
	Data      map[string]interface{} // Additional event data
	Error     error                  // Error (if applicable)
}
