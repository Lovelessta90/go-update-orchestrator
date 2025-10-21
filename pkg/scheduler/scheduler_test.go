package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
	"github.com/dovaclean/go-update-orchestrator/testing/mocks"
)

func TestScheduler_ScheduleImmediate(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	update := core.Update{
		ID:       "immediate-1",
		Name:     "Immediate Update",
		Strategy: core.StrategyImmediate,
	}

	err := scheduler.Schedule(ctx, update)
	if err != nil {
		t.Fatalf("Failed to schedule update: %v", err)
	}

	// Verify update is in pending state
	status, err := scheduler.Status(ctx, "immediate-1")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.Status != core.StatusPending {
		t.Errorf("Expected status %s, got %s", core.StatusPending, status.Status)
	}
}

func TestScheduler_ScheduleScheduled(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	futureTime := time.Now().Add(1 * time.Hour)
	update := core.Update{
		ID:          "scheduled-1",
		Name:        "Scheduled Update",
		Strategy:    core.StrategyScheduled,
		ScheduledAt: &futureTime,
	}

	err := scheduler.Schedule(ctx, update)
	if err != nil {
		t.Fatalf("Failed to schedule update: %v", err)
	}

	// Verify update is in scheduled state
	status, err := scheduler.Status(ctx, "scheduled-1")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.Status != core.StatusScheduled {
		t.Errorf("Expected status %s, got %s", core.StatusScheduled, status.Status)
	}
}

func TestScheduler_ScheduleWithoutID(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	update := core.Update{
		Name:     "No ID Update",
		Strategy: core.StrategyImmediate,
	}

	err := scheduler.Schedule(ctx, update)
	if err == nil {
		t.Error("Expected error when scheduling without ID, got nil")
	}
}

func TestScheduler_ScheduleDuplicate(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	update := core.Update{
		ID:       "duplicate-1",
		Name:     "Duplicate Update",
		Strategy: core.StrategyImmediate,
	}

	// Schedule first time
	err := scheduler.Schedule(ctx, update)
	if err != nil {
		t.Fatalf("Failed to schedule update: %v", err)
	}

	// Try to schedule again
	err = scheduler.Schedule(ctx, update)
	if err == nil {
		t.Error("Expected error when scheduling duplicate update, got nil")
	}
}

func TestScheduler_Cancel(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	update := core.Update{
		ID:       "cancel-1",
		Name:     "Cancelable Update",
		Strategy: core.StrategyImmediate,
	}

	err := scheduler.Schedule(ctx, update)
	if err != nil {
		t.Fatalf("Failed to schedule update: %v", err)
	}

	// Cancel the update
	err = scheduler.Cancel(ctx, "cancel-1")
	if err != nil {
		t.Fatalf("Failed to cancel update: %v", err)
	}

	// Verify update is cancelled
	status, err := scheduler.Status(ctx, "cancel-1")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.Status != core.StatusCancelled {
		t.Errorf("Expected status %s, got %s", core.StatusCancelled, status.Status)
	}
}

func TestScheduler_CancelNonExistent(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	err := scheduler.Cancel(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when cancelling non-existent update, got nil")
	}
}

func TestScheduler_List(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	// Schedule multiple updates with different statuses
	updates := []core.Update{
		{ID: "pending-1", Name: "Pending 1", Strategy: core.StrategyImmediate},
		{ID: "pending-2", Name: "Pending 2", Strategy: core.StrategyImmediate},
		{ID: "scheduled-1", Name: "Scheduled 1", Strategy: core.StrategyScheduled, ScheduledAt: timePtr(time.Now().Add(1 * time.Hour))},
	}

	for _, update := range updates {
		if err := scheduler.Schedule(ctx, update); err != nil {
			t.Fatalf("Failed to schedule update %s: %v", update.ID, err)
		}
	}

	// List pending updates
	pending, err := scheduler.List(ctx, core.StatusPending)
	if err != nil {
		t.Fatalf("Failed to list pending updates: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending updates, got %d", len(pending))
	}

	// List scheduled updates
	scheduled, err := scheduler.List(ctx, core.StatusScheduled)
	if err != nil {
		t.Fatalf("Failed to list scheduled updates: %v", err)
	}

	if len(scheduled) != 1 {
		t.Errorf("Expected 1 scheduled update, got %d", len(scheduled))
	}
}

func TestScheduler_StartStop(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	// Start scheduler
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Try to start again (should fail)
	err = scheduler.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running scheduler, got nil")
	}

	// Stop scheduler
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}

	// Try to stop again (should fail)
	err = scheduler.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running scheduler, got nil")
	}
}

func TestScheduler_IsInUpdateWindow(t *testing.T) {
	scheduler := setupTestScheduler(t)

	tests := []struct {
		name        string
		windowStart *time.Time
		windowEnd   *time.Time
		checkTime   time.Time
		expected    bool
	}{
		{
			name:        "No window specified - always true",
			windowStart: nil,
			windowEnd:   nil,
			checkTime:   time.Now(),
			expected:    true,
		},
		{
			name:        "Only start - after start time",
			windowStart: timePtr(time.Now().Add(-1 * time.Hour)),
			windowEnd:   nil,
			checkTime:   time.Now(),
			expected:    true,
		},
		{
			name:        "Only start - before start time",
			windowStart: timePtr(time.Now().Add(1 * time.Hour)),
			windowEnd:   nil,
			checkTime:   time.Now(),
			expected:    false,
		},
		{
			name:        "Only end - before end time",
			windowStart: nil,
			windowEnd:   timePtr(time.Now().Add(1 * time.Hour)),
			checkTime:   time.Now(),
			expected:    true,
		},
		{
			name:        "Only end - after end time",
			windowStart: nil,
			windowEnd:   timePtr(time.Now().Add(-1 * time.Hour)),
			checkTime:   time.Now(),
			expected:    false,
		},
		{
			name:        "Both - within window",
			windowStart: timePtr(time.Now().Add(-1 * time.Hour)),
			windowEnd:   timePtr(time.Now().Add(1 * time.Hour)),
			checkTime:   time.Now(),
			expected:    true,
		},
		{
			name:        "Both - before window",
			windowStart: timePtr(time.Now().Add(1 * time.Hour)),
			windowEnd:   timePtr(time.Now().Add(2 * time.Hour)),
			checkTime:   time.Now(),
			expected:    false,
		},
		{
			name:        "Both - after window",
			windowStart: timePtr(time.Now().Add(-2 * time.Hour)),
			windowEnd:   timePtr(time.Now().Add(-1 * time.Hour)),
			checkTime:   time.Now(),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := core.Update{
				WindowStart: tt.windowStart,
				WindowEnd:   tt.windowEnd,
			}

			result := scheduler.isInUpdateWindow(update, tt.checkTime)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestScheduler_ScheduledValidation(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	// Scheduled strategy without ScheduledAt should fail
	update := core.Update{
		ID:       "invalid-scheduled",
		Name:     "Invalid Scheduled Update",
		Strategy: core.StrategyScheduled,
		// Missing ScheduledAt
	}

	err := scheduler.Schedule(ctx, update)
	if err == nil {
		t.Error("Expected error for scheduled strategy without ScheduledAt, got nil")
	}
}

func TestScheduler_UnknownStrategy(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	update := core.Update{
		ID:       "unknown-strategy",
		Name:     "Unknown Strategy Update",
		Strategy: "invalid-strategy",
	}

	err := scheduler.Schedule(ctx, update)
	if err == nil {
		t.Error("Expected error for unknown strategy, got nil")
	}
}

func TestScheduler_ProgressivePhases(t *testing.T) {
	scheduler := setupTestScheduler(t)
	ctx := context.Background()

	// Add some test devices to registry
	registry := memory.New()
	for i := 1; i <= 10; i++ {
		device := core.Device{
			ID:     fmt.Sprintf("device-%d", i),
			Name:   fmt.Sprintf("Device %d", i),
			Status: core.DeviceOnline,
		}
		registry.Add(ctx, device)
	}
	scheduler.registry = registry

	// Create progressive update with phases
	update := core.Update{
		ID:       "progressive-1",
		Name:     "Progressive Update",
		Strategy: core.StrategyProgressive,
		RolloutPhases: []core.RolloutPhase{
			{Name: "Canary", Percentage: 10, WaitTime: 100 * time.Millisecond, SuccessRate: 100},
			{Name: "Phase 1", Percentage: 40, WaitTime: 100 * time.Millisecond, SuccessRate: 95},
			{Name: "Phase 2", Percentage: 50, WaitTime: 0, SuccessRate: 90},
		},
	}

	err := scheduler.Schedule(ctx, update)
	if err != nil {
		t.Fatalf("Failed to schedule progressive update: %v", err)
	}

	status, err := scheduler.Status(ctx, "progressive-1")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.Status != core.StatusPending {
		t.Errorf("Expected status %s, got %s", core.StatusPending, status.Status)
	}
}

// Helper functions

func setupTestScheduler(t *testing.T) *Scheduler {
	t.Helper()

	registry := memory.New()
	delivery := mocks.NewMockDelivery()
	orchConfig := orchestrator.DefaultConfig()
	orch, _ := orchestrator.NewDefault(orchConfig, registry, delivery)

	config := DefaultConfig()
	config.TickInterval = 100 * time.Millisecond // Faster for testing

	return New(config, orch, registry)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
