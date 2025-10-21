package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

func TestSQLiteRegistry_AddAndGet(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()
	now := time.Now()

	device := core.Device{
		ID:              "device-1",
		Name:            "Test Device",
		Address:         "192.168.1.100",
		Status:          core.DeviceOnline,
		LastSeen:        &now,
		FirmwareVersion: "v1.0.0",
		Location:        "New York",
		Metadata: map[string]string{
			"region": "us-east",
			"type":   "pos",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add device
	err := registry.Add(ctx, device)
	if err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}

	// Get device
	retrieved, err := registry.Get(ctx, "device-1")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	// Verify fields
	if retrieved.ID != device.ID {
		t.Errorf("Expected ID %s, got %s", device.ID, retrieved.ID)
	}
	if retrieved.Name != device.Name {
		t.Errorf("Expected Name %s, got %s", device.Name, retrieved.Name)
	}
	if retrieved.Status != device.Status {
		t.Errorf("Expected Status %s, got %s", device.Status, retrieved.Status)
	}
	if retrieved.Metadata["region"] != "us-east" {
		t.Errorf("Expected metadata region us-east, got %s", retrieved.Metadata["region"])
	}
}

func TestSQLiteRegistry_Update(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()
	now := time.Now()

	device := core.Device{
		ID:              "device-1",
		Name:            "Original Name",
		Address:         "192.168.1.100",
		Status:          core.DeviceOnline,
		FirmwareVersion: "v1.0.0",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Add device
	if err := registry.Add(ctx, device); err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}

	// Update device
	device.Name = "Updated Name"
	device.Status = core.DeviceOffline
	device.FirmwareVersion = "v2.0.0"

	if err := registry.Update(ctx, device); err != nil {
		t.Fatalf("Failed to update device: %v", err)
	}

	// Get updated device
	retrieved, err := registry.Get(ctx, "device-1")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrieved.Name)
	}
	if retrieved.Status != core.DeviceOffline {
		t.Errorf("Expected status offline, got %s", retrieved.Status)
	}
	if retrieved.FirmwareVersion != "v2.0.0" {
		t.Errorf("Expected firmware v2.0.0, got %s", retrieved.FirmwareVersion)
	}
}

func TestSQLiteRegistry_Delete(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	device := core.Device{
		ID:      "device-1",
		Name:    "Test Device",
		Address: "192.168.1.100",
		Status:  core.DeviceOnline,
	}

	// Add device
	if err := registry.Add(ctx, device); err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}

	// Delete device
	if err := registry.Delete(ctx, "device-1"); err != nil {
		t.Fatalf("Failed to delete device: %v", err)
	}

	// Verify device is gone
	_, err := registry.Get(ctx, "device-1")
	if err != core.ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

func TestSQLiteRegistry_List_FilterByStatus(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	// Add multiple devices
	devices := []core.Device{
		{ID: "device-1", Name: "Device 1", Address: "addr1", Status: core.DeviceOnline},
		{ID: "device-2", Name: "Device 2", Address: "addr2", Status: core.DeviceOffline},
		{ID: "device-3", Name: "Device 3", Address: "addr3", Status: core.DeviceOnline},
	}

	for _, device := range devices {
		if err := registry.Add(ctx, device); err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}
	}

	// Filter by online status
	online := core.DeviceOnline
	filter := core.Filter{Status: &online}

	results, err := registry.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 online devices, got %d", len(results))
	}

	for _, device := range results {
		if device.Status != core.DeviceOnline {
			t.Errorf("Expected online device, got %s", device.Status)
		}
	}
}

func TestSQLiteRegistry_List_FilterByLocation(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	// Add devices in different locations
	devices := []core.Device{
		{ID: "device-1", Name: "Device 1", Address: "addr1", Status: core.DeviceOnline, Location: "New York"},
		{ID: "device-2", Name: "Device 2", Address: "addr2", Status: core.DeviceOnline, Location: "London"},
		{ID: "device-3", Name: "Device 3", Address: "addr3", Status: core.DeviceOnline, Location: "New York"},
	}

	for _, device := range devices {
		if err := registry.Add(ctx, device); err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}
	}

	// Filter by location
	filter := core.Filter{Location: "New York"}

	results, err := registry.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 devices in New York, got %d", len(results))
	}

	for _, device := range results {
		if device.Location != "New York" {
			t.Errorf("Expected location New York, got %s", device.Location)
		}
	}
}

func TestSQLiteRegistry_List_FilterByMetadata(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	// Add devices with different metadata
	devices := []core.Device{
		{
			ID: "device-1", Name: "Device 1", Address: "addr1", Status: core.DeviceOnline,
			Metadata: map[string]string{"region": "us-east", "type": "pos"},
		},
		{
			ID: "device-2", Name: "Device 2", Address: "addr2", Status: core.DeviceOnline,
			Metadata: map[string]string{"region": "us-west", "type": "pos"},
		},
		{
			ID: "device-3", Name: "Device 3", Address: "addr3", Status: core.DeviceOnline,
			Metadata: map[string]string{"region": "us-east", "type": "kiosk"},
		},
	}

	for _, device := range devices {
		if err := registry.Add(ctx, device); err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}
	}

	// Filter by metadata tags
	filter := core.Filter{
		Tags: map[string]string{
			"region": "us-east",
			"type":   "pos",
		},
	}

	results, err := registry.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 device matching metadata, got %d", len(results))
	}

	if results[0].ID != "device-1" {
		t.Errorf("Expected device-1, got %s", results[0].ID)
	}
}

func TestSQLiteRegistry_List_Pagination(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	// Add 10 devices
	for i := 1; i <= 10; i++ {
		device := core.Device{
			ID:      core.Device{}.ID + string(rune('0'+i)),
			Name:    "Device",
			Address: "addr",
			Status:  core.DeviceOnline,
		}
		device.ID = "device-" + string(rune('0'+i))
		if err := registry.Add(ctx, device); err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}
	}

	// Get first page (limit 5)
	filter := core.Filter{Limit: 5, Offset: 0}
	page1, err := registry.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list page 1: %v", err)
	}

	if len(page1) != 5 {
		t.Errorf("Expected 5 devices in page 1, got %d", len(page1))
	}

	// Get second page
	filter.Offset = 5
	page2, err := registry.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list page 2: %v", err)
	}

	if len(page2) != 5 {
		t.Errorf("Expected 5 devices in page 2, got %d", len(page2))
	}

	// Ensure pages don't overlap
	for _, d1 := range page1 {
		for _, d2 := range page2 {
			if d1.ID == d2.ID {
				t.Errorf("Device %s appears in both pages", d1.ID)
			}
		}
	}
}

func TestSQLiteRegistry_UpdateNonExistent(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	device := core.Device{
		ID:      "nonexistent",
		Name:    "Test",
		Address: "addr",
		Status:  core.DeviceOnline,
	}

	err := registry.Update(ctx, device)
	if err != core.ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

func TestSQLiteRegistry_DeleteNonExistent(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	err := registry.Delete(ctx, "nonexistent")
	if err != core.ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

func TestSQLiteRegistry_GetNonExistent(t *testing.T) {
	registry := setupTestRegistry(t)
	defer cleanup(registry)

	ctx := context.Background()

	_, err := registry.Get(ctx, "nonexistent")
	if err != core.ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

// Helper functions

func setupTestRegistry(t *testing.T) *Registry {
	t.Helper()

	// Create temp database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	registry, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test registry: %v", err)
	}

	return registry
}

func cleanup(registry *Registry) {
	registry.Close()
}
