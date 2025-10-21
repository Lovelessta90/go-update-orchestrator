package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	httpdelivery "github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
	sshdelivery "github.com/dovaclean/go-update-orchestrator/pkg/delivery/ssh"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/sqlite"
	"github.com/dovaclean/go-update-orchestrator/pkg/scheduler"
	"github.com/dovaclean/go-update-orchestrator/web"
)

func main() {
	fmt.Println("🚀 Go Update Orchestrator - Demo Application")
	fmt.Println("============================================")
	fmt.Println()

	ctx := context.Background()

	// Choose registry type (SQLite for persistence, Memory for testing)
	useSQLite := false // Set to true to use SQLite persistence
	var reg interface {
		core.Registry
		interface{ Close() error }
	}

	if useSQLite {
		fmt.Println("📦 Initializing SQLite registry...")
		sqliteReg, err := sqlite.New("orchestrator.db")
		if err != nil {
			log.Fatalf("Failed to create SQLite registry: %v", err)
		}
		reg = sqliteReg
		defer reg.Close()
		fmt.Println("   ✓ SQLite registry initialized (orchestrator.db)")
	} else {
		fmt.Println("📦 Initializing in-memory registry...")
		memReg := memory.New()
		reg = struct {
			core.Registry
			interface{ Close() error }
		}{
			Registry: memReg,
			Close:    func() error { return nil },
		}
		fmt.Println("   ✓ Memory registry initialized")
	}

	// Add sample devices
	fmt.Println("\n📱 Adding sample devices...")
	sampleDevices := []core.Device{
		{
			ID:              "pos-001",
			Name:            "POS Terminal - Store 1",
			Address:         "192.168.1.10",
			Status:          core.DeviceOnline,
			FirmwareVersion: "v1.0.0",
			Location:        "New York",
			Metadata: map[string]string{
				"region":   "us-east",
				"type":     "pos",
				"priority": "high",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:              "pos-002",
			Name:            "POS Terminal - Store 2",
			Address:         "192.168.1.11",
			Status:          core.DeviceOnline,
			FirmwareVersion: "v1.0.0",
			Location:        "Los Angeles",
			Metadata: map[string]string{
				"region":   "us-west",
				"type":     "pos",
				"priority": "medium",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:              "pos-003",
			Name:            "POS Terminal - Store 3",
			Address:         "192.168.1.12",
			Status:          core.DeviceOffline,
			FirmwareVersion: "v0.9.0",
			Location:        "Chicago",
			Metadata: map[string]string{
				"region":   "us-central",
				"type":     "pos",
				"priority": "low",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:              "kiosk-001",
			Name:            "Information Kiosk",
			Address:         "192.168.2.10",
			Status:          core.DeviceOnline,
			FirmwareVersion: "v2.1.0",
			Location:        "New York",
			Metadata: map[string]string{
				"region": "us-east",
				"type":   "kiosk",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:              "kiosk-002",
			Name:            "Self-Service Kiosk",
			Address:         "192.168.2.11",
			Status:          core.DeviceOnline,
			FirmwareVersion: "v2.0.0",
			Location:        "Boston",
			Metadata: map[string]string{
				"region": "us-east",
				"type":   "kiosk",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, device := range sampleDevices {
		if err := reg.Add(ctx, device); err != nil {
			log.Printf("Warning: Failed to add device %s: %v", device.ID, err)
		} else {
			fmt.Printf("   ✓ Added %s (%s) - %s\n", device.ID, device.Name, device.Status)
		}
	}

	// Choose delivery mechanism
	fmt.Println("\n🚚 Initializing delivery mechanism...")

	// For demo, use HTTP delivery (easier to test without SSH setup)
	httpConfig := httpdelivery.DefaultConfig()
	httpConfig.Timeout = 10 * time.Second
	delivery := httpdelivery.NewWithConfig(httpConfig)
	fmt.Println("   ✓ HTTP delivery initialized")

	// If you have SSH setup, you can use this instead:
	/*
	sshConfig := sshdelivery.DefaultConfig()
	sshConfig.PrivateKeyPath = "/home/user/.ssh/id_rsa"
	sshConfig.RemotePath = "/tmp/update.bin"
	delivery := sshdelivery.NewWithConfig(sshConfig)
	fmt.Println("   ✓ SSH delivery initialized")
	*/

	// Initialize orchestrator
	fmt.Println("\n🎯 Initializing orchestrator...")
	orchConfig := orchestrator.DefaultConfig()
	orchConfig.MaxConcurrent = 10
	orch, err := orchestrator.NewDefault(orchConfig, reg, delivery)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}
	fmt.Println("   ✓ Orchestrator initialized")

	// Initialize scheduler
	fmt.Println("\n⏰ Initializing scheduler...")
	schedConfig := scheduler.DefaultConfig()
	schedConfig.TickInterval = 30 * time.Second
	schedConfig.MaxConcurrentUpdates = 3
	sched := scheduler.New(schedConfig, orch, reg)

	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()
	fmt.Println("   ✓ Scheduler started")

	// Schedule some example updates
	fmt.Println("\n📅 Scheduling example updates...")

	// Example 1: Immediate update for POS devices
	immediateUpdate := core.Update{
		ID:       "update-immediate-001",
		Name:     "Critical Security Patch",
		Strategy: core.StrategyImmediate,
		DeviceFilter: &core.Filter{
			Tags: map[string]string{"type": "pos"},
		},
		CreatedAt: time.Now(),
	}

	if err := sched.Schedule(ctx, immediateUpdate); err != nil {
		log.Printf("Warning: Failed to schedule immediate update: %v", err)
	} else {
		fmt.Println("   ✓ Scheduled immediate update for POS devices")
	}

	// Example 2: Scheduled update for future
	futureTime := time.Now().Add(2 * time.Minute)
	scheduledUpdate := core.Update{
		ID:          "update-scheduled-001",
		Name:        "Firmware Update v2.0",
		Strategy:    core.StrategyScheduled,
		ScheduledAt: &futureTime,
		WindowStart: timePtr(time.Now().Add(1 * time.Minute)),
		WindowEnd:   timePtr(time.Now().Add(5 * time.Minute)),
		DeviceFilter: &core.Filter{
			Location: "New York",
		},
		CreatedAt: time.Now(),
	}

	if err := sched.Schedule(ctx, scheduledUpdate); err != nil {
		log.Printf("Warning: Failed to schedule future update: %v", err)
	} else {
		fmt.Printf("   ✓ Scheduled update for %s (in 2 minutes)\n", futureTime.Format("15:04:05"))
	}

	// Example 3: Progressive rollout
	progressiveUpdate := core.Update{
		ID:       "update-progressive-001",
		Name:     "Gradual Feature Rollout",
		Strategy: core.StrategyProgressive,
		RolloutPhases: []core.RolloutPhase{
			{Name: "Canary", Percentage: 20, WaitTime: 30 * time.Second, SuccessRate: 100},
			{Name: "Phase 1", Percentage: 40, WaitTime: 30 * time.Second, SuccessRate: 95},
			{Name: "Phase 2", Percentage: 40, WaitTime: 0, SuccessRate: 90},
		},
		DeviceFilter: &core.Filter{},
		CreatedAt:    time.Now(),
	}

	if err := sched.Schedule(ctx, progressiveUpdate); err != nil {
		log.Printf("Warning: Failed to schedule progressive update: %v", err)
	} else {
		fmt.Println("   ✓ Scheduled progressive rollout (Canary → Phase 1 → Phase 2)")
	}

	// Start web UI
	fmt.Println("\n🌐 Starting Web UI...")
	webConfig := web.DefaultConfig()
	webConfig.Address = ":8080"

	server, err := web.New(webConfig, orch, sched, reg)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// Print access information
	fmt.Println("\n" + separator())
	fmt.Println("✅ Update Orchestrator is running!")
	fmt.Println(separator())
	fmt.Println()
	fmt.Println("🌐 Web Interface:")
	fmt.Println("   Dashboard:  http://localhost:8080")
	fmt.Println("   Devices:    http://localhost:8080/devices")
	fmt.Println("   Updates:    http://localhost:8080/updates")
	fmt.Println()
	fmt.Println("📡 API Endpoints:")
	fmt.Println("   GET  /api/devices         - List all devices")
	fmt.Println("   GET  /api/devices/{id}    - Get device details")
	fmt.Println("   GET  /api/updates         - List all updates")
	fmt.Println("   GET  /api/updates/{id}    - Get update status")
	fmt.Println("   POST /api/updates/schedule - Schedule new update")
	fmt.Println("   POST /api/updates/cancel   - Cancel update")
	fmt.Println()
	fmt.Println("📊 Current Status:")
	fmt.Printf("   Devices:    %d (3 online, 2 offline)\n", len(sampleDevices))
	fmt.Printf("   Updates:    3 scheduled\n")
	fmt.Printf("   Registry:   %s\n", registryType(useSQLite))
	fmt.Printf("   Delivery:   HTTP\n")
	fmt.Println()
	fmt.Println(separator())
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println(separator())

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\n🛑 Shutting down...")
	fmt.Println("   ✓ Stopping scheduler...")
	sched.Stop()
	fmt.Println("   ✓ Closing registry...")
	reg.Close()
	fmt.Println("\n👋 Goodbye!")
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func separator() string {
	return "============================================"
}

func registryType(useSQLite bool) string {
	if useSQLite {
		return "SQLite (persistent)"
	}
	return "Memory (temporary)"
}
