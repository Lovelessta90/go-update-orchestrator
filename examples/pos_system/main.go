package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
)

// DeviceJSON represents the JSON structure for device data.
type DeviceJSON struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Address         string            `json:"address"`
	Status          string            `json:"status"`
	FirmwareVersion string            `json:"firmware_version"`
	Location        string            `json:"location"`
	Metadata        map[string]string `json:"metadata"`
}

func main() {
	fmt.Println("==========================================================")
	fmt.Println("  POS System Update Orchestrator - Real-World Scenarios")
	fmt.Println("==========================================================")
	fmt.Println()

	// Load devices from JSON file
	devices := loadDevices("devices.json")
	registry := setupRegistry(devices)

	// Demonstrate different update scenarios
	fmt.Println("Available Update Scenarios:")
	fmt.Println("  1. Immediate Update (Online Devices Only)")
	fmt.Println("  2. Scheduled Maintenance Window (2 AM - 4 AM)")
	fmt.Println("  3. Progressive Rollout (Canary → 50% → 100%)")
	fmt.Println("  4. On-Connect Strategy (Offline Devices)")
	fmt.Println("  5. Region-Based Update (Northeast stores)")
	fmt.Println("  6. Firmware Version Update (v2.1.x → v2.4.0)")
	fmt.Println()

	// Scenario 1: Immediate Update
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 1: Immediate Update to Online Devices")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioImmediate(registry)

	// Scenario 2: Scheduled Maintenance Window
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 2: Scheduled Maintenance Window (2 AM - 4 AM)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioScheduled(registry)

	// Scenario 3: Progressive Rollout
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 3: Progressive Rollout (Risk Mitigation)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioProgressive(registry)

	// Scenario 4: On-Connect Strategy
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 4: On-Connect Strategy (Offline Devices)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioOnConnect(registry)

	// Scenario 5: Region-Based Update
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 5: Region-Based Update (Northeast Stores)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioRegionBased(registry)

	// Scenario 6: Firmware Version Update
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SCENARIO 6: Upgrade Old Firmware (v2.0.x, v2.1.x)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	scenarioFirmwareUpgrade(registry)
}

func loadDevices(filename string) []DeviceJSON {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read devices file: %v", err)
	}

	var devices []DeviceJSON
	if err := json.Unmarshal(data, &devices); err != nil {
		log.Fatalf("Failed to parse devices JSON: %v", err)
	}

	return devices
}

func setupRegistry(devicesJSON []DeviceJSON) *memory.Registry {
	registry := memory.New()
	ctx := context.Background()
	now := time.Now()

	for _, d := range devicesJSON {
		var lastSeen *time.Time
		if d.Status == "online" {
			t := now.Add(-time.Minute * 5) // Online devices seen 5 min ago
			lastSeen = &t
		} else if d.Status == "offline" {
			t := now.Add(-time.Hour * 24) // Offline devices seen 24h ago
			lastSeen = &t
		}

		device := core.Device{
			ID:              d.ID,
			Name:            d.Name,
			Address:         d.Address,
			Status:          core.DeviceStatus(d.Status),
			LastSeen:        lastSeen,
			FirmwareVersion: d.FirmwareVersion,
			Location:        d.Location,
			Metadata:        d.Metadata,
			CreatedAt:       now.Add(-time.Hour * 24 * 365), // Registered 1 year ago
			UpdatedAt:       now,
		}

		if err := registry.Add(ctx, device); err != nil {
			log.Fatalf("Failed to add device %s: %v", d.ID, err)
		}
	}

	return registry
}

func scenarioImmediate(registry *memory.Registry) {
	ctx := context.Background()

	// Filter for online devices only
	onlineStatus := core.DeviceOnline
	filter := core.Filter{
		Status: &onlineStatus,
	}

	devices, err := registry.List(ctx, filter)
	if err != nil {
		log.Printf("Error listing devices: %v", err)
		return
	}

	fmt.Printf("Use Case: Critical security patch needs immediate deployment\n")
	fmt.Printf("Strategy: Push to all ONLINE devices RIGHT NOW\n\n")

	update := core.Update{
		ID:         "update-immediate-001",
		Name:       "Security Patch v2.4.1 (CVE-2024-1234)",
		PayloadURL: "https://cdn.retailcorp.com/updates/pos-security-patch-v2.4.1.bin",
		Strategy:   core.StrategyImmediate,
		CreatedAt:  time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Strategy: %s\n", update.Strategy)
	fmt.Printf("Payload: %s\n\n", update.PayloadURL)

	fmt.Printf("Devices that will receive update immediately:\n")
	for _, device := range devices {
		fmt.Printf("  ✓ %s (v%s) - %s - Last seen: %s\n",
			device.Name,
			device.FirmwareVersion,
			device.Location,
			device.LastSeen.Format("15:04:05"))
	}

	fmt.Printf("\nTotal: %d devices will be updated NOW\n", len(devices))
	fmt.Printf("Note: %d offline devices will NOT receive this update\n", getTotalDevices(registry)-len(devices))
	fmt.Printf("      → Consider on-connect strategy for offline devices\n")
}

func scenarioScheduled(registry *memory.Registry) {
	ctx := context.Background()

	// Schedule for tonight at 2 AM
	tomorrow := time.Now().Add(24 * time.Hour)
	scheduledTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 2, 0, 0, 0, time.Local)
	windowEnd := scheduledTime.Add(2 * time.Hour) // 2 AM - 4 AM

	fmt.Printf("Use Case: Major firmware update during low-traffic hours\n")
	fmt.Printf("Strategy: Execute during maintenance window\n\n")

	update := core.Update{
		ID:          "update-scheduled-001",
		Name:        "Firmware v2.5.0 - Feature Release",
		PayloadURL:  "https://cdn.retailcorp.com/updates/pos-firmware-v2.5.0.bin",
		Strategy:    core.StrategyScheduled,
		ScheduledAt: &scheduledTime,
		WindowStart: &scheduledTime,
		WindowEnd:   &windowEnd,
		CreatedAt:   time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Strategy: %s\n", update.Strategy)
	fmt.Printf("Scheduled: %s\n", scheduledTime.Format("Mon Jan 2, 2006 at 3:04 PM MST"))
	fmt.Printf("Window: %s - %s\n\n", scheduledTime.Format("3:04 PM"), windowEnd.Format("3:04 PM MST"))

	allDevices, _ := registry.List(ctx, core.Filter{})

	fmt.Printf("Execution Plan:\n")
	fmt.Printf("  1. At %s, orchestrator wakes up\n", scheduledTime.Format("2:00 AM"))
	fmt.Printf("  2. Check device status (online/offline)\n")
	fmt.Printf("  3. Push updates to online devices\n")
	fmt.Printf("  4. Offline devices will update when they come online\n")
	fmt.Printf("  5. Window closes at %s\n\n", windowEnd.Format("4:00 AM"))

	fmt.Printf("All registered devices (%d total):\n", len(allDevices))
	for _, device := range allDevices {
		symbol := "●"
		statusStr := ""
		switch device.Status {
		case core.DeviceOnline:
			statusStr = "Will update at 2 AM"
		case core.DeviceOffline:
			statusStr = "Will update when online"
		case core.DeviceUnknown:
			statusStr = "Status unknown, will try"
		}
		fmt.Printf("  %s %s - %s (%s)\n", symbol, device.Name, device.Location, statusStr)
	}

	fmt.Printf("\nBenefit: Updates happen during low-traffic hours, minimal disruption\n")
}

func scenarioProgressive(registry *memory.Registry) {
	ctx := context.Background()

	fmt.Printf("Use Case: New firmware with unknown stability\n")
	fmt.Printf("Strategy: Gradual rollout to minimize risk\n\n")

	// Define rollout phases
	phases := []core.RolloutPhase{
		{
			Name:        "Canary (Test Stores)",
			Percentage:  10, // 10% of devices
			WaitTime:    24 * time.Hour,
			SuccessRate: 95, // Must be 95%+ successful
		},
		{
			Name:        "Phase 1 (50%)",
			Percentage:  50,
			WaitTime:    12 * time.Hour,
			SuccessRate: 90,
		},
		{
			Name:        "Phase 2 (100%)",
			Percentage:  100,
			WaitTime:    0,
			SuccessRate: 85,
		},
	}

	update := core.Update{
		ID:            "update-progressive-001",
		Name:          "Firmware v3.0.0 - Major Release",
		PayloadURL:    "https://cdn.retailcorp.com/updates/pos-firmware-v3.0.0.bin",
		Strategy:      core.StrategyProgressive,
		RolloutPhases: phases,
		CreatedAt:     time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Strategy: %s\n\n", update.Strategy)

	allDevices, _ := registry.List(ctx, core.Filter{})
	totalDevices := len(allDevices)

	fmt.Printf("Rollout Plan (%d total devices):\n\n", totalDevices)

	for i, phase := range phases {
		deviceCount := (totalDevices * phase.Percentage) / 100
		fmt.Printf("Phase %d: %s\n", i+1, phase.Name)
		fmt.Printf("  • Devices: %d (%d%%)\n", deviceCount, phase.Percentage)
		fmt.Printf("  • Success threshold: %d%%\n", phase.SuccessRate)
		if phase.WaitTime > 0 {
			fmt.Printf("  • Wait time: %v before next phase\n", phase.WaitTime)
		}
		fmt.Printf("  • Action: Monitor success rate, proceed if >= %d%%\n", phase.SuccessRate)
		fmt.Printf("\n")
	}

	fmt.Printf("Safety Features:\n")
	fmt.Printf("  • Automatic rollback if success rate drops below threshold\n")
	fmt.Printf("  • Human approval required between phases (optional)\n")
	fmt.Printf("  • Real-time monitoring of each phase\n")
	fmt.Printf("  • Can pause/cancel at any phase\n")
}

func scenarioOnConnect(registry *memory.Registry) {
	ctx := context.Background()

	// Filter for offline devices
	offlineStatus := core.DeviceOffline
	filter := core.Filter{
		Status: &offlineStatus,
	}

	offlineDevices, err := registry.List(ctx, filter)
	if err != nil {
		log.Printf("Error listing devices: %v", err)
		return
	}

	fmt.Printf("Use Case: Fleet includes intermittently connected devices\n")
	fmt.Printf("Strategy: Update devices when they connect\n\n")

	update := core.Update{
		ID:         "update-onconnect-001",
		Name:       "Firmware v2.4.0 - Persistent Update",
		PayloadURL: "https://cdn.retailcorp.com/updates/pos-firmware-v2.4.0.bin",
		Strategy:   core.StrategyOnConnect,
		CreatedAt:  time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Strategy: %s\n\n", update.Strategy)

	fmt.Printf("How it works:\n")
	fmt.Printf("  1. Update is registered in the system\n")
	fmt.Printf("  2. When a device connects (heartbeat), orchestrator checks for pending updates\n")
	fmt.Printf("  3. Update is pushed immediately upon connection\n")
	fmt.Printf("  4. No manual intervention required\n\n")

	fmt.Printf("Currently offline devices (%d):\n", len(offlineDevices))
	for _, device := range offlineDevices {
		lastSeenStr := "never"
		if device.LastSeen != nil {
			lastSeenStr = fmt.Sprintf("%v ago", time.Since(*device.LastSeen).Round(time.Hour))
		}
		fmt.Printf("  ⏸ %s - Last seen: %s\n", device.Name, lastSeenStr)
	}

	fmt.Printf("\nBenefit: No devices left behind, updates happen automatically\n")
	fmt.Printf("Real-world: Perfect for mobile POS, kiosks, seasonal locations\n")
}

func scenarioRegionBased(registry *memory.Registry) {
	ctx := context.Background()

	// Filter for northeast region
	filter := core.Filter{
		Tags: map[string]string{
			"region": "northeast",
		},
	}

	// Note: The memory registry doesn't implement tag filtering yet,
	// so we'll filter manually for this example
	allDevices, _ := registry.List(ctx, core.Filter{})
	var northeastDevices []core.Device
	for _, device := range allDevices {
		if region, ok := device.Metadata["region"]; ok && region == "northeast" {
			northeastDevices = append(northeastDevices, device)
		}
	}

	fmt.Printf("Use Case: Regional rollout for compliance or testing\n")
	fmt.Printf("Strategy: Target specific geographic region\n\n")

	update := core.Update{
		ID:           "update-region-001",
		Name:         "Tax Calculation Update - Northeast States",
		PayloadURL:   "https://cdn.retailcorp.com/updates/tax-update-northeast.bin",
		Strategy:     core.StrategyImmediate,
		DeviceFilter: &filter,
		CreatedAt:    time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Target: Northeast region only\n\n")

	fmt.Printf("Matched devices (%d):\n", len(northeastDevices))
	for _, device := range northeastDevices {
		fmt.Printf("  ✓ %s - %s (Priority: %s)\n",
			device.Name,
			device.Location,
			device.Metadata["priority"])
	}

	fmt.Printf("\nOther use cases for filtering:\n")
	fmt.Printf("  • Priority-based: High-priority stores first\n")
	fmt.Printf("  • Store type: Flagship vs standard\n")
	fmt.Printf("  • Firmware version: Only devices on v2.x\n")
	fmt.Printf("  • Custom tags: Beta testers, etc.\n")
}

func scenarioFirmwareUpgrade(registry *memory.Registry) {
	ctx := context.Background()

	allDevices, _ := registry.List(ctx, core.Filter{})

	// Find devices with old firmware (manually filter)
	var oldDevices []core.Device
	for _, device := range allDevices {
		// Check if firmware version starts with "2.0." or "2.1."
		if len(device.FirmwareVersion) >= 4 {
			prefix := device.FirmwareVersion[:4]
			if prefix == "2.0." || prefix == "2.1." {
				oldDevices = append(oldDevices, device)
			}
		}
	}

	fmt.Printf("Use Case: Upgrade devices running old firmware\n")
	fmt.Printf("Strategy: Target specific firmware versions\n\n")

	update := core.Update{
		ID:         "update-firmware-001",
		Name:       "Firmware Upgrade: v2.0.x/v2.1.x → v2.4.0",
		PayloadURL: "https://cdn.retailcorp.com/updates/pos-firmware-v2.4.0.bin",
		Strategy:   core.StrategyScheduled,
		CreatedAt:  time.Now(),
	}

	fmt.Printf("Update: %s\n", update.Name)
	fmt.Printf("Target: Devices with firmware < v2.2.0\n\n")

	fmt.Printf("Devices needing upgrade (%d):\n", len(oldDevices))
	for _, device := range oldDevices {
		age := "unknown"
		if device.LastSeen != nil {
			age = time.Since(*device.LastSeen).Round(time.Hour).String()
		}
		fmt.Printf("  ⚠ %s - Current: v%s → Target: v2.4.0 (Last seen: %s)\n",
			device.Name,
			device.FirmwareVersion,
			age)
	}

	fmt.Printf("\nBenefit: Keep fleet standardized, easier support\n")
	fmt.Printf("Filter capabilities:\n")
	fmt.Printf("  • MinFirmware: Only devices >= v2.0.0\n")
	fmt.Printf("  • MaxFirmware: Only devices < v2.3.0\n")
	fmt.Printf("  • Ensures controlled, predictable updates\n")
}

func getTotalDevices(registry *memory.Registry) int {
	devices, _ := registry.List(context.Background(), core.Filter{})
	return len(devices)
}
