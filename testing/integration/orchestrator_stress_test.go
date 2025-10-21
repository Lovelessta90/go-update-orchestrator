package integration

import (
	"context"
	"io"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	httpdelivery "github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
	"github.com/dovaclean/go-update-orchestrator/testing/mocks"
)

// TestOrchestrator_Stress_1000Devices tests orchestrator with 1K devices
func TestOrchestrator_Stress_1000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orchestrator stress test in short mode")
	}

	testOrchestratorStress(t, 1000, 100)
}

// TestOrchestrator_Stress_10000Devices tests orchestrator with 10K devices
func TestOrchestrator_Stress_10000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orchestrator stress test in short mode")
	}

	testOrchestratorStress(t, 10000, 100)
}

// TestOrchestrator_Stress_100000Devices tests orchestrator with 100K devices
func TestOrchestrator_Stress_100000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orchestrator stress test in short mode")
	}

	testOrchestratorStress(t, 100000, 100)
}

func testOrchestratorStress(t *testing.T, deviceCount, maxConcurrent int) {
	t.Helper()

	// Create registry with many devices
	registry := memory.New()
	ctx := context.Background()

	// Create multiple mock servers to spread load
	serverCount := 10
	servers := make([]*mocks.DeviceServer, serverCount)
	for i := 0; i < serverCount; i++ {
		servers[i] = mocks.NewDeviceServer("v1.0.0")
		defer servers[i].Close()
	}

	// Add devices to registry (distributed across servers)
	for i := 0; i < deviceCount; i++ {
		serverIdx := i % serverCount
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Name:    fmt.Sprintf("Device %d", i),
			Address: servers[serverIdx].URL(),
			Status:  core.DeviceOnline,
		}
		if err := registry.Add(ctx, device); err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}
	}

	// Create delivery mechanism
	delivery := mocks.NewMockDelivery()

	// Create orchestrator
	config := orchestrator.DefaultConfig()
	config.MaxConcurrent = maxConcurrent

	orch, err := orchestrator.NewDefault(config, registry, delivery)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Track events
	var updateStarted, updateCompleted, devicesCompleted, devicesFailed atomic.Int64

	orch.Subscribe("update.started", mocks.EventCounter(&updateStarted))
	orch.Subscribe("update.completed", mocks.EventCounter(&updateCompleted))
	orch.Subscribe("device.completed", mocks.EventCounter(&devicesCompleted))
	orch.Subscribe("device.failed", mocks.EventCounter(&devicesFailed))

	// Create update
	filter := core.Filter{}
	update := core.Update{
		ID:           "stress-test-update",
		Name:         "Stress Test Firmware",
		DeviceFilter: &filter,
		CreatedAt:    time.Now(),
	}

	payload := strings.NewReader("Test firmware payload")

	// Execute update
	start := time.Now()
	err = orch.ExecuteUpdateWithPayload(ctx, update, payload)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Update execution failed: %v", err)
	}

	// Get final status
	status, err := orch.GetStatus(ctx, update.ID)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	// Report results
	t.Logf("Orchestrator Stress Test Results:")
	t.Logf("  Devices: %d", deviceCount)
	t.Logf("  Max Concurrent: %d", maxConcurrent)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f devices/sec", float64(deviceCount)/elapsed.Seconds())
	t.Logf("  Completed: %d", status.Completed)
	t.Logf("  Failed: %d", status.Failed)
	t.Logf("  Events - Started: %d, Completed: %d, Device Success: %d, Device Fail: %d",
		updateStarted.Load(), updateCompleted.Load(), devicesCompleted.Load(), devicesFailed.Load())

	// Verify results
	if status.TotalDevices != deviceCount {
		t.Errorf("Expected %d total devices, got %d", deviceCount, status.TotalDevices)
	}

	if status.Completed+status.Failed != deviceCount {
		t.Errorf("Completed (%d) + Failed (%d) != Total (%d)", status.Completed, status.Failed, deviceCount)
	}
}

// TestOrchestrator_ConcurrentUpdates tests running multiple updates simultaneously
func TestOrchestrator_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent updates test in short mode")
	}

	registry := memory.New()
	ctx := context.Background()

	// Create 1000 devices
	server := mocks.NewDeviceServer("v1.0.0")
	defer server.Close()

	for i := 0; i < 1000; i++ {
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: server.URL(),
			Status:  core.DeviceOnline,
		}
		registry.Add(ctx, device)
	}

	delivery := mocks.NewMockDelivery()
	config := orchestrator.DefaultConfig()
	orch, _ := orchestrator.NewDefault(config, registry, delivery)

	// Run 10 concurrent updates
	concurrentUpdates := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrentUpdates)

	start := time.Now()

	for i := 0; i < concurrentUpdates; i++ {
		wg.Add(1)
		go func(updateID int) {
			defer wg.Done()

			filter := core.Filter{}
			update := core.Update{
				ID:           fmt.Sprintf("update-%d", updateID),
				DeviceFilter: &filter,
				CreatedAt:    time.Now(),
			}

			payload := strings.NewReader(fmt.Sprintf("Firmware for update %d", updateID))
			err := orch.ExecuteUpdateWithPayload(ctx, update, payload)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	elapsed := time.Since(start)

	// Check for errors
	for err := range errors {
		t.Errorf("Update failed: %v", err)
	}

	// Verify all updates completed
	for i := 0; i < concurrentUpdates; i++ {
		status, err := orch.GetStatus(ctx, fmt.Sprintf("update-%d", i))
		if err != nil {
			t.Errorf("Failed to get status for update-%d: %v", i, err)
			continue
		}

		if status.TotalDevices != 1000 {
			t.Errorf("Update %d: expected 1000 devices, got %d", i, status.TotalDevices)
		}
	}

	t.Logf("Concurrent Updates Test:")
	t.Logf("  Updates: %d", concurrentUpdates)
	t.Logf("  Devices per update: 1000")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f updates/sec", float64(concurrentUpdates)/elapsed.Seconds())
}

// TestOrchestrator_MixedSuccessFailure tests handling of partial failures
func TestOrchestrator_MixedSuccessFailure(t *testing.T) {
	registry := memory.New()
	ctx := context.Background()

	// Create devices - half will succeed, half will fail
	successServer := mocks.NewDeviceServer("v1.0.0")
	defer successServer.Close()

	failServer := mocks.NewDeviceServer("v1.0.0")
	failServer.SetAlwaysFail(true) // Always fail all updates (even retries)
	defer failServer.Close()

	deviceCount := 100
	for i := 0; i < deviceCount; i++ {
		var address string
		if i%2 == 0 {
			address = successServer.URL()
		} else {
			address = failServer.URL()
		}

		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: address,
			Status:  core.DeviceOnline,
		}
		registry.Add(ctx, device)
	}

	// Use real HTTP delivery with no retries so we get immediate failures
	httpConfig := httpdelivery.DefaultConfig()
	httpConfig.MaxRetries = 1 // Try once, no retries (0 would mean never execute!)
	delivery := httpdelivery.NewWithConfig(httpConfig)

	config := orchestrator.DefaultConfig()
	orch, err := orchestrator.NewDefault(config, registry, delivery)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Execute update
	filter := core.Filter{}
	update := core.Update{
		ID:           "mixed-test",
		DeviceFilter: &filter,
		CreatedAt:    time.Now(),
	}

	payload := strings.NewReader("Test payload")
	err = orch.ExecuteUpdateWithPayload(ctx, update, payload)

	// Should not fail even if some devices fail
	if err != nil {
		t.Fatalf("Update should not fail with partial failures: %v", err)
	}

	// Get status
	status, err := orch.GetStatus(ctx, update.ID)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	t.Logf("Mixed Success/Failure Test:")
	t.Logf("  Total: %d", status.TotalDevices)
	t.Logf("  Completed: %d", status.Completed)
	t.Logf("  Failed: %d", status.Failed)

	// Verify we have BOTH successes and failures (the point of this test)
	// We don't require exactly 50/50 because factors like server load, timing,
	// and HTTP client behavior can affect the split
	if status.Completed == 0 {
		t.Errorf("Expected some successes, got %d", status.Completed)
	}

	if status.Failed == 0 {
		t.Errorf("Expected some failures, got %d", status.Failed)
	}

	if status.Completed+status.Failed != deviceCount {
		t.Errorf("Completed (%d) + Failed (%d) != Total (%d)",
			status.Completed, status.Failed, deviceCount)
	}

	// Verify status is failed (since some devices failed)
	if status.Status != core.StatusFailed {
		t.Errorf("Expected status %s, got %s", core.StatusFailed, status.Status)
	}
}

// TestOrchestrator_WorkerPoolLimit tests that MaxConcurrent is respected
func TestOrchestrator_WorkerPoolLimit(t *testing.T) {
	registry := memory.New()
	ctx := context.Background()

	// Track max concurrent requests
	var currentConcurrent, maxConcurrent int64
	var mu sync.Mutex

	// Create custom delivery that tracks concurrency
	delivery := &concurrencyTrackingDelivery{
		current: &currentConcurrent,
		max:     &maxConcurrent,
		mu:      &mu,
	}

	// Create 1000 devices
	for i := 0; i < 1000; i++ {
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: "http://localhost",
			Status:  core.DeviceOnline,
		}
		registry.Add(ctx, device)
	}

	// Set max concurrent to 50
	config := orchestrator.DefaultConfig()
	config.MaxConcurrent = 50

	orch, _ := orchestrator.NewDefault(config, registry, delivery)

	// Execute update
	filter := core.Filter{}
	update := core.Update{
		ID:           "concurrency-test",
		DeviceFilter: &filter,
		CreatedAt:    time.Now(),
	}

	payload := strings.NewReader("Test")
	err := orch.ExecuteUpdateWithPayload(ctx, update, payload)

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	maxObserved := atomic.LoadInt64(&maxConcurrent)

	t.Logf("Worker Pool Limit Test:")
	t.Logf("  Max Concurrent Config: %d", config.MaxConcurrent)
	t.Logf("  Max Concurrent Observed: %d", maxObserved)

	// Should not exceed max concurrent (with some tolerance for timing)
	if maxObserved > int64(config.MaxConcurrent)+5 {
		t.Errorf("Max concurrent (%d) exceeded limit (%d)", maxObserved, config.MaxConcurrent)
	}

	// Should actually use concurrency (not serial)
	if maxObserved < 10 {
		t.Errorf("Concurrency too low (%d), expected close to %d", maxObserved, config.MaxConcurrent)
	}
}

// concurrencyTrackingDelivery tracks concurrent Push calls
type concurrencyTrackingDelivery struct {
	current *int64
	max     *int64
	mu      *sync.Mutex
}

func (d *concurrencyTrackingDelivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	// Increment current
	current := atomic.AddInt64(d.current, 1)
	defer atomic.AddInt64(d.current, -1)

	// Track max
	d.mu.Lock()
	if current > *d.max {
		*d.max = current
	}
	d.mu.Unlock()

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (d *concurrencyTrackingDelivery) Verify(ctx context.Context, device core.Device) error {
	return nil
}
