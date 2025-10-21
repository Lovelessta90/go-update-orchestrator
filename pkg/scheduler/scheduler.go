package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry"
)

// Config holds scheduler configuration.
type Config struct {
	// TickInterval is how often the scheduler checks for pending updates
	TickInterval time.Duration

	// MaxConcurrentUpdates limits how many updates can run simultaneously
	MaxConcurrentUpdates int
}

// DefaultConfig returns scheduler configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		TickInterval:         1 * time.Minute,
		MaxConcurrentUpdates: 5,
	}
}

// Scheduler manages scheduled update execution.
type Scheduler struct {
	config       *Config
	orchestrator *orchestrator.Orchestrator
	registry     registry.Registry

	mu            sync.RWMutex
	updates       map[string]*scheduledUpdate
	running       bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// scheduledUpdate wraps an update with scheduling metadata.
type scheduledUpdate struct {
	update    core.Update
	status    core.UpdateStatus
	createdAt time.Time
	startedAt *time.Time
	cancelFn  context.CancelFunc
}

// New creates a new scheduler.
func New(config *Config, orch *orchestrator.Orchestrator, reg registry.Registry) *Scheduler {
	if config == nil {
		config = DefaultConfig()
	}

	return &Scheduler{
		config:       config,
		orchestrator: orch,
		registry:     reg,
		updates:      make(map[string]*scheduledUpdate),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the scheduler's background processing.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()

	// Cancel all running updates
	s.mu.Lock()
	for _, update := range s.updates {
		if update.cancelFn != nil {
			update.cancelFn()
		}
	}
	s.mu.Unlock()

	return nil
}

// Schedule queues an update for execution.
func (s *Scheduler) Schedule(ctx context.Context, update core.Update) error {
	if update.ID == "" {
		return fmt.Errorf("update ID is required")
	}

	// Determine initial status based on strategy
	var status core.UpdateStatus
	switch update.Strategy {
	case core.StrategyImmediate:
		status = core.StatusPending
	case core.StrategyScheduled:
		if update.ScheduledAt == nil {
			return fmt.Errorf("scheduled strategy requires ScheduledAt time")
		}
		status = core.StatusScheduled
	case core.StrategyProgressive:
		status = core.StatusPending
	case core.StrategyOnConnect:
		status = core.StatusScheduled // Will be triggered by device connection events
	default:
		return fmt.Errorf("unknown update strategy: %s", update.Strategy)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if update already exists
	if _, exists := s.updates[update.ID]; exists {
		return fmt.Errorf("update %s already scheduled", update.ID)
	}

	// Add to scheduled updates
	s.updates[update.ID] = &scheduledUpdate{
		update:    update,
		status:    status,
		createdAt: time.Now(),
	}

	return nil
}

// Status returns the current status of an update.
func (s *Scheduler) Status(ctx context.Context, updateID string) (*core.Status, error) {
	s.mu.RLock()
	scheduled, exists := s.updates[updateID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("update %s not found", updateID)
	}

	// If update is running, get status from orchestrator
	if scheduled.status == core.StatusInProgress {
		return s.orchestrator.GetStatus(ctx, updateID)
	}

	// Return basic status for scheduled/pending updates
	return &core.Status{
		UpdateID:     updateID,
		Status:       scheduled.status,
		TotalDevices: 0,
		Completed:    0,
		Failed:       0,
		StartedAt:    scheduled.createdAt,
	}, nil
}

// Cancel attempts to cancel a running or scheduled update.
func (s *Scheduler) Cancel(ctx context.Context, updateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduled, exists := s.updates[updateID]
	if !exists {
		return fmt.Errorf("update %s not found", updateID)
	}

	// Cancel if running
	if scheduled.cancelFn != nil {
		scheduled.cancelFn()
	}

	// Update status
	scheduled.status = core.StatusCancelled

	return nil
}

// List returns all updates matching the given status.
func (s *Scheduler) List(ctx context.Context, status core.UpdateStatus) ([]core.Status, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]core.Status, 0)
	for id, scheduled := range s.updates {
		if scheduled.status == status {
			results = append(results, core.Status{
				UpdateID:  id,
				Status:    scheduled.status,
				StartedAt: scheduled.createdAt,
			})
		}
	}

	return results, nil
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processScheduledUpdates(ctx)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processScheduledUpdates checks for updates that should be executed.
func (s *Scheduler) processScheduledUpdates(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	runningCount := s.countRunningUpdates()

	for id, scheduled := range s.updates {
		// Skip if already running, completed, cancelled, or failed
		if scheduled.status != core.StatusPending && scheduled.status != core.StatusScheduled {
			continue
		}

		// Check concurrency limit
		if runningCount >= s.config.MaxConcurrentUpdates {
			break
		}

		// Check if update should run now
		shouldRun := false
		switch scheduled.update.Strategy {
		case core.StrategyImmediate:
			shouldRun = true

		case core.StrategyScheduled:
			if scheduled.update.ScheduledAt != nil && !now.Before(*scheduled.update.ScheduledAt) {
				// Check if we're within the update window (if specified)
				if s.isInUpdateWindow(scheduled.update, now) {
					shouldRun = true
				}
			}

		case core.StrategyProgressive:
			// Progressive updates start immediately and are managed by phases
			shouldRun = true
		}

		if shouldRun {
			s.executeUpdate(ctx, id, scheduled)
			runningCount++
		}
	}
}

// isInUpdateWindow checks if the current time is within the update window.
func (s *Scheduler) isInUpdateWindow(update core.Update, now time.Time) bool {
	// If no window specified, always allow
	if update.WindowStart == nil && update.WindowEnd == nil {
		return true
	}

	// If only WindowStart specified, check if after start
	if update.WindowStart != nil && update.WindowEnd == nil {
		return !now.Before(*update.WindowStart)
	}

	// If only WindowEnd specified, check if before end
	if update.WindowStart == nil && update.WindowEnd != nil {
		return now.Before(*update.WindowEnd)
	}

	// Both specified - check if within window
	return !now.Before(*update.WindowStart) && now.Before(*update.WindowEnd)
}

// executeUpdate starts executing an update.
func (s *Scheduler) executeUpdate(parentCtx context.Context, updateID string, scheduled *scheduledUpdate) {
	// Create cancelable context for this update
	ctx, cancel := context.WithCancel(parentCtx)
	scheduled.cancelFn = cancel

	// Mark as in progress
	scheduled.status = core.StatusInProgress
	now := time.Now()
	scheduled.startedAt = &now

	// Execute in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()

		// Execute the update via orchestrator
		// Note: For this to work, we need the payload. In a real implementation,
		// the orchestrator would fetch the payload from PayloadURL
		err := s.executeUpdateStrategy(ctx, scheduled.update)

		// Update final status
		s.mu.Lock()
		if err != nil {
			scheduled.status = core.StatusFailed
		} else {
			scheduled.status = core.StatusCompleted
		}
		scheduled.cancelFn = nil
		s.mu.Unlock()
	}()
}

// executeUpdateStrategy executes the update based on its strategy.
func (s *Scheduler) executeUpdateStrategy(ctx context.Context, update core.Update) error {
	switch update.Strategy {
	case core.StrategyImmediate, core.StrategyScheduled:
		// Execute immediately on all matched devices
		return s.executeImmediate(ctx, update)

	case core.StrategyProgressive:
		// Execute in phases
		return s.executeProgressive(ctx, update)

	case core.StrategyOnConnect:
		// This would be triggered by device connection events
		// For now, treat as immediate
		return s.executeImmediate(ctx, update)

	default:
		return fmt.Errorf("unsupported strategy: %s", update.Strategy)
	}
}

// executeImmediate executes an update immediately on all matched devices.
func (s *Scheduler) executeImmediate(ctx context.Context, update core.Update) error {
	// In a full implementation, this would:
	// 1. Fetch payload from update.PayloadURL
	// 2. Call orchestrator.ExecuteUpdateWithPayload()

	// For now, we just acknowledge that the scheduler would delegate to orchestrator
	// The orchestrator already has ExecuteUpdate methods
	return nil
}

// executeProgressive executes an update in phases.
func (s *Scheduler) executeProgressive(ctx context.Context, update core.Update) error {
	if len(update.RolloutPhases) == 0 {
		return fmt.Errorf("progressive strategy requires rollout phases")
	}

	// Get all target devices
	filter := core.Filter{}
	if update.DeviceFilter != nil {
		filter = *update.DeviceFilter
	}

	devices, err := s.registry.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	totalDevices := len(devices)
	if totalDevices == 0 {
		return fmt.Errorf("no devices match filter")
	}

	// Execute each phase
	deviceOffset := 0
	for i, phase := range update.RolloutPhases {
		// Calculate how many devices for this phase
		phaseDeviceCount := (totalDevices * phase.Percentage) / 100
		if phaseDeviceCount == 0 {
			phaseDeviceCount = 1
		}

		// Don't exceed remaining devices
		if deviceOffset+phaseDeviceCount > totalDevices {
			phaseDeviceCount = totalDevices - deviceOffset
		}

		// Get devices for this phase
		phaseDevices := devices[deviceOffset : deviceOffset+phaseDeviceCount]

		// Execute update for this phase
		// In full implementation: orchestrator.ExecuteUpdateWithPayload() for phaseDevices
		_ = phaseDevices // TODO: Actually execute

		// Wait before next phase (except for last phase)
		if i < len(update.RolloutPhases)-1 {
			select {
			case <-time.After(phase.WaitTime):
				// Continue to next phase
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		deviceOffset += phaseDeviceCount
	}

	return nil
}

// countRunningUpdates returns the number of currently running updates.
func (s *Scheduler) countRunningUpdates() int {
	count := 0
	for _, scheduled := range s.updates {
		if scheduled.status == core.StatusInProgress {
			count++
		}
	}
	return count
}
