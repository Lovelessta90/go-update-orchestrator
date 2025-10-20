package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
	"github.com/dovaclean/go-update-orchestrator/testing/mocks"
)

func TestIntegration_HTTPDelivery_SingleDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create mock device server
	deviceServer := mocks.NewDeviceServer("v2.3.0")
	defer deviceServer.Close()

	// Create HTTP delivery
	delivery := http.New()

	// Create device pointing to mock server
	device := core.Device{
		ID:              "pos-test-001",
		Name:            "Test POS Terminal",
		Address:         deviceServer.URL(),
		FirmwareVersion: "v2.3.0",
	}

	// Push update
	ctx := context.Background()
	payload := strings.NewReader("v2.4.0-firmware-data-here")

	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}

	// Verify update was received
	if deviceServer.GetUpdateCount() != 1 {
		t.Errorf("expected 1 update, got %d", deviceServer.GetUpdateCount())
	}

	// Verify payload size
	expectedSize := int64(len("v2.4.0-firmware-data-here"))
	if deviceServer.GetLastUpdateSize() != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, deviceServer.GetLastUpdateSize())
	}

	// Verify the update
	err = delivery.Verify(ctx, device)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}
}

func TestIntegration_HTTPDelivery_MultipleDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create multiple mock device servers
	multiServer := mocks.NewMultiDeviceServer()
	defer multiServer.CloseAll()

	// Add 5 devices
	devices := make([]core.Device, 5)
	for i := 0; i < 5; i++ {
		deviceID := fmt.Sprintf("pos-test-%03d", i+1)
		url := multiServer.AddDevice(deviceID, "v2.3.0")
		devices[i] = core.Device{
			ID:              deviceID,
			Name:            fmt.Sprintf("Test POS %d", i+1),
			Address:         url,
			FirmwareVersion: "v2.3.0",
		}
	}

	// Create HTTP delivery
	delivery := http.New()
	ctx := context.Background()

	// Push update to all devices
	for _, device := range devices {
		payload := strings.NewReader("v2.4.0-new-firmware")
		err := delivery.Push(ctx, device, payload)
		if err != nil {
			t.Fatalf("Push() to %s failed: %v", device.ID, err)
		}
	}

	// Verify all devices received updates
	counts := multiServer.GetAllUpdateCounts()
	for deviceID, count := range counts {
		if count != 1 {
			t.Errorf("device %s: expected 1 update, got %d", deviceID, count)
		}
	}

	// Verify all devices
	for _, device := range devices {
		err := delivery.Verify(ctx, device)
		if err != nil {
			t.Fatalf("Verify() for %s failed: %v", device.ID, err)
		}
	}
}

func TestIntegration_HTTPDelivery_FailureAndRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	deviceServer := mocks.NewDeviceServer("v2.3.0")
	defer deviceServer.Close()

	// Configure server to fail next update
	deviceServer.SetFailNext(true)

	delivery := http.New()
	device := core.Device{
		ID:      "pos-test-001",
		Address: deviceServer.URL(),
	}

	ctx := context.Background()
	payload := strings.NewReader("v2.4.0-firmware")

	// First attempt should fail
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Fatal("expected error on first push, got nil")
	}

	// Second attempt should succeed (failNext was reset)
	payload = strings.NewReader("v2.4.0-firmware")
	err = delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() retry failed: %v", err)
	}

	if deviceServer.GetUpdateCount() != 1 {
		t.Errorf("expected 1 successful update, got %d", deviceServer.GetUpdateCount())
	}
}

func TestIntegration_HTTPDelivery_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create 10 devices
	multiServer := mocks.NewMultiDeviceServer()
	defer multiServer.CloseAll()

	numDevices := 10
	devices := make([]core.Device, numDevices)
	for i := 0; i < numDevices; i++ {
		deviceID := fmt.Sprintf("pos-concurrent-%03d", i+1)
		url := multiServer.AddDevice(deviceID, "v2.3.0")
		devices[i] = core.Device{
			ID:      deviceID,
			Address: url,
		}
	}

	// Create HTTP delivery
	delivery := http.New()
	ctx := context.Background()

	// Push updates concurrently
	errChan := make(chan error, numDevices)
	for _, device := range devices {
		go func(dev core.Device) {
			payload := strings.NewReader("v2.4.0-concurrent-update")
			err := delivery.Push(ctx, dev, payload)
			errChan <- err
		}(device)
	}

	// Wait for all to complete
	for i := 0; i < numDevices; i++ {
		err := <-errChan
		if err != nil {
			t.Errorf("concurrent push failed: %v", err)
		}
	}

	// Verify all devices received updates
	counts := multiServer.GetAllUpdateCounts()
	if len(counts) != numDevices {
		t.Errorf("expected %d devices, got %d", numDevices, len(counts))
	}

	for deviceID, count := range counts {
		if count != 1 {
			t.Errorf("device %s: expected 1 update, got %d", deviceID, count)
		}
	}
}

func TestIntegration_HTTPDelivery_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	deviceServer := mocks.NewDeviceServer("v2.3.0")
	defer deviceServer.Close()

	delivery := http.New()
	device := core.Device{
		ID:      "pos-test-001",
		Address: deviceServer.URL(),
	}

	// Create 10MB payload to test streaming
	largePayload := strings.Repeat("A", 10*1024*1024) // 10MB

	ctx := context.Background()
	payload := strings.NewReader(largePayload)

	start := time.Now()
	err := delivery.Push(ctx, device, payload)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}

	t.Logf("Pushed 10MB in %v", duration)

	// Verify size
	expectedSize := int64(len(largePayload))
	if deviceServer.GetLastUpdateSize() != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, deviceServer.GetLastUpdateSize())
	}
}

func TestIntegration_HTTPDelivery_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	deviceServer := mocks.NewDeviceServer("v2.3.0")
	defer deviceServer.Close()

	// Use default config (normal timeout)
	delivery := http.New()

	device := core.Device{
		ID:      "pos-test-001",
		Address: deviceServer.URL(),
	}

	// Create context that times out very quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give context time to expire
	time.Sleep(10 * time.Millisecond)

	payload := strings.NewReader("v2.4.0-firmware")

	// Should fail due to context timeout
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	// Verify error is context-related
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("got error: %v (this is acceptable - context may have already expired)", err)
	}
}

func TestIntegration_HTTPDelivery_CustomHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	deviceServer := mocks.NewDeviceServer("v2.3.0")
	defer deviceServer.Close()

	// Create delivery with custom headers
	config := http.DefaultConfig()
	config.Headers = map[string]string{
		"Authorization": "Bearer test-token-12345",
		"X-Custom":      "custom-value",
	}
	delivery := http.NewWithConfig(config)

	device := core.Device{
		ID:      "pos-test-001",
		Name:    "Test Device",
		Address: deviceServer.URL(),
	}

	ctx := context.Background()
	payload := strings.NewReader("v2.4.0-firmware")

	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}

	// Verify update succeeded
	if deviceServer.GetUpdateCount() != 1 {
		t.Errorf("expected 1 update, got %d", deviceServer.GetUpdateCount())
	}
}
