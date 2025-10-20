package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
	"github.com/dovaclean/go-update-orchestrator/pkg/events"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
)

func main() {
	fmt.Println("Go Update Orchestrator - Simple Example")
	fmt.Println("========================================")

	// Create registry and add sample devices
	registry := memory.New()
	devices := []core.Device{
		{ID: "device-1", Name: "POS Terminal 1", Address: "https://device1.example.com"},
		{ID: "device-2", Name: "POS Terminal 2", Address: "https://device2.example.com"},
		{ID: "device-3", Name: "POS Terminal 3", Address: "https://device3.example.com"},
	}

	ctx := context.Background()
	for _, device := range devices {
		if err := registry.Add(ctx, device); err != nil {
			log.Fatalf("Failed to add device: %v", err)
		}
		fmt.Printf("Added device: %s\n", device.Name)
	}

	// Create HTTP delivery mechanism
	delivery := http.New()

	// Create orchestrator with configuration
	config := orchestrator.DefaultConfig()
	config.MaxConcurrent = 2 // Update 2 devices at a time

	orch, err := orchestrator.New(config, registry, delivery)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Subscribe to events
	orch.Subscribe(events.EventUpdateStarted, events.HandlerFunc(func(ctx context.Context, event events.Event) {
		fmt.Printf("[EVENT] Update started: %s\n", event.UpdateID)
	}))

	orch.Subscribe(events.EventDeviceCompleted, events.HandlerFunc(func(ctx context.Context, event events.Event) {
		fmt.Printf("[EVENT] Device completed: %s (Update: %s)\n", event.DeviceID, event.UpdateID)
	}))

	orch.Subscribe(events.EventUpdateCompleted, events.HandlerFunc(func(ctx context.Context, event events.Event) {
		fmt.Printf("[EVENT] Update completed: %s\n", event.UpdateID)
	}))

	// Create update job
	update := core.Update{
		ID:         "update-001",
		Name:       "Firmware v2.0",
		PayloadURL: "https://updates.example.com/firmware-v2.0.bin",
		DeviceIDs:  []string{"device-1", "device-2", "device-3"},
		CreatedAt:  time.Now(),
	}

	fmt.Printf("\nExecuting update: %s\n", update.Name)
	fmt.Printf("Target devices: %d\n", len(update.DeviceIDs))
	fmt.Printf("Max concurrent: %d\n\n", config.MaxConcurrent)

	// Execute update (this would actually run in real implementation)
	if err := orch.ExecuteUpdate(ctx, update); err != nil {
		log.Fatalf("Update failed: %v", err)
	}

	fmt.Println("\nUpdate execution initiated successfully!")
	fmt.Println("\nNote: This is a skeleton example. Full implementation coming soon.")
}
