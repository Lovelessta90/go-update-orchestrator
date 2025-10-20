package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

func TestNew(t *testing.T) {
	tracker := New()
	if tracker == nil {
		t.Fatal("New() returned nil")
	}
	if tracker.updates == nil {
		t.Fatal("tracker.updates is nil")
	}
}

func TestTracker_StartAndGetProgress(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	totalDevices := 10

	// Start tracking
	tracker.Start(ctx, updateID, totalDevices)

	// Get progress
	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if prog.UpdateID != updateID {
		t.Errorf("Expected updateID %s, got %s", updateID, prog.UpdateID)
	}

	if prog.TotalDevices != totalDevices {
		t.Errorf("Expected %d total devices, got %d", totalDevices, prog.TotalDevices)
	}

	if prog.CompletedDevices != 0 {
		t.Errorf("Expected 0 completed devices, got %d", prog.CompletedDevices)
	}

	if prog.EstimatedEnd != nil {
		t.Error("Expected no estimated end time initially")
	}
}

func TestTracker_UpdateDevice(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	tracker.Start(ctx, updateID, 3)

	// Update device to in_progress
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusInProgress), 1024)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if prog.InProgressDevices != 1 {
		t.Errorf("Expected 1 in-progress device, got %d", prog.InProgressDevices)
	}

	if prog.BytesTransferred != 1024 {
		t.Errorf("Expected 1024 bytes transferred, got %d", prog.BytesTransferred)
	}

	// Check device progress
	deviceProg, exists := prog.DeviceProgress["device-1"]
	if !exists {
		t.Fatal("Device progress not found for device-1")
	}

	if deviceProg.Status != core.StatusInProgress {
		t.Errorf("Expected status %s, got %s", core.StatusInProgress, deviceProg.Status)
	}

	if deviceProg.BytesTransferred != 1024 {
		t.Errorf("Expected 1024 bytes, got %d", deviceProg.BytesTransferred)
	}
}

func TestTracker_DeviceCompletion(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	tracker.Start(ctx, updateID, 2)

	// Device 1: in_progress → completed
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusInProgress), 1024)
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusCompleted), 2048)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if prog.CompletedDevices != 1 {
		t.Errorf("Expected 1 completed device, got %d", prog.CompletedDevices)
	}

	if prog.InProgressDevices != 0 {
		t.Errorf("Expected 0 in-progress devices, got %d", prog.InProgressDevices)
	}

	// Check device has end time
	deviceProg := prog.DeviceProgress["device-1"]
	if deviceProg.EndTime == nil {
		t.Error("Expected end time to be set for completed device")
	}
}

func TestTracker_DeviceFailure(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	tracker.Start(ctx, updateID, 2)

	// Device fails
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusInProgress), 512)
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusFailed), 512)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if prog.FailedDevices != 1 {
		t.Errorf("Expected 1 failed device, got %d", prog.FailedDevices)
	}

	if prog.InProgressDevices != 0 {
		t.Errorf("Expected 0 in-progress devices, got %d", prog.InProgressDevices)
	}
}

func TestTracker_Complete(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	tracker.Start(ctx, updateID, 2)

	// Complete some devices
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusCompleted), 1024)
	tracker.UpdateDevice(ctx, updateID, "device-2", string(core.StatusCompleted), 2048)

	// Mark update as complete
	tracker.Complete(ctx, updateID)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if prog.CompletedDevices != 2 {
		t.Errorf("Expected 2 completed devices, got %d", prog.CompletedDevices)
	}
}

func TestTracker_EstimatedEndTime(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	totalDevices := 10
	tracker.Start(ctx, updateID, totalDevices)

	// Complete 5 devices
	for i := 0; i < 5; i++ {
		deviceID := "device-" + string(rune('0'+i))
		tracker.UpdateDevice(ctx, updateID, deviceID, string(core.StatusCompleted), 1024)
	}

	// Small delay to ensure time progresses
	time.Sleep(10 * time.Millisecond)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	// Should have estimated end time since we're 50% done
	if prog.EstimatedEnd == nil {
		t.Error("Expected estimated end time when partially complete")
	}

	if prog.EstimatedEnd != nil {
		// Estimated end should be in the future
		if prog.EstimatedEnd.Before(time.Now()) {
			t.Error("Estimated end time should be in the future")
		}
	}
}

func TestTracker_MultipleUpdates(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	// Track two updates simultaneously
	tracker.Start(ctx, "update-1", 5)
	tracker.Start(ctx, "update-2", 10)

	tracker.UpdateDevice(ctx, "update-1", "device-a", string(core.StatusCompleted), 1024)
	tracker.UpdateDevice(ctx, "update-2", "device-b", string(core.StatusCompleted), 2048)

	// Check update-1
	prog1, err := tracker.GetProgress(ctx, "update-1")
	if err != nil {
		t.Fatalf("GetProgress failed for update-1: %v", err)
	}

	if prog1.CompletedDevices != 1 {
		t.Errorf("update-1: expected 1 completed device, got %d", prog1.CompletedDevices)
	}

	if prog1.BytesTransferred != 1024 {
		t.Errorf("update-1: expected 1024 bytes, got %d", prog1.BytesTransferred)
	}

	// Check update-2
	prog2, err := tracker.GetProgress(ctx, "update-2")
	if err != nil {
		t.Fatalf("GetProgress failed for update-2: %v", err)
	}

	if prog2.CompletedDevices != 1 {
		t.Errorf("update-2: expected 1 completed device, got %d", prog2.CompletedDevices)
	}

	if prog2.BytesTransferred != 2048 {
		t.Errorf("update-2: expected 2048 bytes, got %d", prog2.BytesTransferred)
	}
}

func TestTracker_GetProgress_NotFound(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	_, err := tracker.GetProgress(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent update")
	}
}

func TestTracker_UpdateDevice_UnknownUpdate(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	// Should not panic when updating unknown update
	tracker.UpdateDevice(ctx, "unknown", "device-1", string(core.StatusCompleted), 1024)

	// Verify it doesn't create the update
	_, err := tracker.GetProgress(ctx, "unknown")
	if err == nil {
		t.Error("Expected error, update should not be created by UpdateDevice")
	}
}

func TestTracker_BytesAccumulation(t *testing.T) {
	tracker := New()
	ctx := context.Background()

	updateID := "update-123"
	tracker.Start(ctx, updateID, 1)

	// Update device multiple times with increasing bytes
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusInProgress), 1024)
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusInProgress), 2048)
	tracker.UpdateDevice(ctx, updateID, "device-1", string(core.StatusCompleted), 4096)

	prog, err := tracker.GetProgress(ctx, updateID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	// Total bytes = 1024 + 2048 + 4096
	expectedBytes := int64(1024 + 2048 + 4096)
	if prog.BytesTransferred != expectedBytes {
		t.Errorf("Expected %d total bytes, got %d", expectedBytes, prog.BytesTransferred)
	}
}
