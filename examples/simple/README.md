# Simple Example

This example demonstrates the basic usage of Go Update Orchestrator.

## What it does

1. Creates an in-memory device registry
2. Adds three sample devices
3. Configures the orchestrator with max 2 concurrent updates
4. Subscribes to update events
5. Executes an update job across all devices

## Running

```bash
cd examples/simple
go run main.go
```

## Expected Output

```
Go Update Orchestrator - Simple Example
========================================
Added device: POS Terminal 1
Added device: POS Terminal 2
Added device: POS Terminal 3

Executing update: Firmware v2.0
Target devices: 3
Max concurrent: 2

[EVENT] Update started: update-001
[EVENT] Device completed: device-1 (Update: update-001)
[EVENT] Device completed: device-2 (Update: update-001)
[EVENT] Device completed: device-3 (Update: update-001)
[EVENT] Update completed: update-001

Update execution initiated successfully!
```

## Key Concepts

- **Registry**: Stores device information
- **Delivery**: Handles protocol-specific update delivery
- **Orchestrator**: Coordinates all components
- **Events**: Subscribe to progress updates
- **Config**: Control concurrency and retry behavior
