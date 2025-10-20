package core

import "time"

// UpdateStatus represents the state of an update job.
type UpdateStatus string

const (
	StatusPending    UpdateStatus = "pending"     // Waiting to start
	StatusScheduled  UpdateStatus = "scheduled"   // Scheduled for future execution
	StatusInProgress UpdateStatus = "in_progress" // Currently executing
	StatusCompleted  UpdateStatus = "completed"   // Successfully completed
	StatusFailed     UpdateStatus = "failed"      // Failed (some/all devices)
	StatusCancelled  UpdateStatus = "cancelled"   // Cancelled by user
	StatusPaused     UpdateStatus = "paused"      // Temporarily paused
)

// UpdateStrategy defines how the update should be rolled out.
type UpdateStrategy string

const (
	StrategyImmediate   UpdateStrategy = "immediate"   // Push to all online devices now
	StrategyScheduled   UpdateStrategy = "scheduled"   // Execute at specific time
	StrategyProgressive UpdateStrategy = "progressive" // Gradual rollout in phases
	StrategyOnConnect   UpdateStrategy = "on_connect"  // Update when device connects
)

// Update represents an update job to be executed.
type Update struct {
	ID          string            // Unique update identifier
	Name        string            // Human-readable name
	PayloadURL  string            // Location of the update payload
	DeviceIDs   []string          // Target devices (if empty, use DeviceFilter)
	DeviceFilter *Filter          // Dynamic device selection
	Strategy    UpdateStrategy    // How to roll out the update
	ScheduledAt *time.Time        // When to execute (for scheduled strategy)
	WindowStart *time.Time        // Start of update window (e.g., 2 AM)
	WindowEnd   *time.Time        // End of update window (e.g., 4 AM)
	RolloutPhases []RolloutPhase  // Phases for progressive strategy
	Metadata    map[string]string // Custom update metadata
	CreatedAt   time.Time         // When the update was created
}

// RolloutPhase represents a phase in a progressive rollout.
type RolloutPhase struct {
	Name        string    // Phase name (e.g., "Canary", "Phase 1")
	Percentage  int       // Percentage of devices to update (1-100)
	WaitTime    time.Duration // Time to wait after phase before next
	SuccessRate int       // Minimum success rate to proceed (0-100)
}

// Status represents the current state of an update job.
type Status struct {
	UpdateID      string            // Update identifier
	Status        UpdateStatus      // Overall status
	TotalDevices  int               // Total number of target devices
	Completed     int               // Number of devices completed
	Failed        int               // Number of devices failed
	InProgress    int               // Number of devices currently updating
	DeviceStatus  map[string]string // Per-device status
	StartedAt     time.Time         // When the update started
	CompletedAt   *time.Time        // When the update completed (nil if not done)
	EstimatedEnd  *time.Time        // Estimated completion time
}
