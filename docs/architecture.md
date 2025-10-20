# Architecture

## Overview

Go Update Orchestrator uses an event-driven, interface-based architecture designed for maximum flexibility and scalability.

## Core Principles

### 1. Interface-Based Design
All major components are defined by interfaces, allowing:
- Easy mocking for testing
- Multiple implementations (e.g., HTTP vs SSH delivery)
- Pluggable components without core changes

### 2. Event-Driven Communication
Components communicate via events instead of direct coupling:
- Loose coupling between components
- Easy to add new event handlers
- Better observability and monitoring

### 3. Bounded Concurrency
Worker pools ensure predictable resource usage:
- Configurable max concurrent updates
- Prevents memory/connection exhaustion
- Fair distribution of work

### 4. Streaming Updates
Payloads are streamed, never fully loaded:
- Constant memory usage regardless of file size
- Can handle multi-GB firmware updates
- Uses buffer pools for zero allocations

### 5. Context-Based Cancellation
All operations support context cancellation:
- Graceful shutdown
- Timeout handling
- Proper resource cleanup

## Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                      Orchestrator                       │
│  (Coordinates all components, main entry point)         │
└─────────────────────────────────────────────────────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
┌──────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│   Registry   │ │ Delivery │ │  Events  │ │   Progress   │
│              │ │          │ │          │ │              │
│ - Memory     │ │ - HTTP   │ │ - Bus    │ │ - Tracker    │
│ - SQLite     │ │ - SSH    │ │ - Types  │ │ - Estimator  │
└──────────────┘ └──────────┘ └──────────┘ └──────────────┘
         │              │              │              │
         └──────────────┴──────────────┴──────────────┘
                          │
                          ▼
                 ┌─────────────────┐
                 │    Internal     │
                 │                 │
                 │ - Worker Pool   │
                 │ - Retry Logic   │
                 │ - Buffer Pool   │
                 │ - Validation    │
                 └─────────────────┘
```

## Data Flow

### Update Execution Flow

1. **Client submits update** → Orchestrator.ExecuteUpdate()
2. **Validation** → Validate update and configuration
3. **Device lookup** → Registry.List() fetches target devices
4. **Event emission** → EventUpdateStarted published
5. **Worker pool creation** → Create bounded worker pool
6. **Parallel execution**:
   - For each device (concurrently):
     - Fetch payload (streaming)
     - Delivery.Push() sends to device
     - Emit EventDeviceStarted
     - Track progress
     - Retry on failure
     - Emit EventDeviceCompleted/Failed
7. **Completion** → EventUpdateCompleted published
8. **Status tracking** → Progress.GetProgress() available throughout

## Key Design Decisions

### Why Interfaces?
- **Flexibility**: Easy to swap implementations
- **Testing**: Mock all dependencies
- **Extensibility**: Add custom delivery/registry without core changes

### Why Event Bus?
- **Decoupling**: Components don't depend on each other
- **Observability**: Easy to add monitoring/logging
- **Extensibility**: Add handlers without changing core

### Why Worker Pools?
- **Resource control**: Prevent connection/memory exhaustion
- **Predictability**: Consistent resource usage
- **Efficiency**: Reuse goroutines instead of spawning thousands

### Why Streaming?
- **Scalability**: Handle any payload size
- **Efficiency**: Constant memory usage
- **Performance**: No disk I/O for buffering

## Performance Characteristics

### Memory
- **Constant**: O(max_concurrent * buffer_size)
- **Independent of**: Payload size, device count
- **Typical**: ~100MB for 100 concurrent updates with 1MB buffers

### Concurrency
- **Configurable**: MaxConcurrent setting
- **Bounded**: Never exceeds configured limit
- **Fair**: Round-robin task distribution

### Network
- **Streaming**: Payloads streamed, not buffered
- **Parallel**: Multiple devices updated concurrently
- **Retry**: Exponential backoff for transient failures

## Extension Points

### Custom Delivery Mechanism
Implement the `delivery.Delivery` interface:
```go
type MyDelivery struct {}

func (d *MyDelivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
    // Your custom delivery logic
}

func (d *MyDelivery) Verify(ctx context.Context, device core.Device) error {
    // Your custom verification logic
}
```

### Custom Registry
Implement the `registry.Registry` interface for different storage backends.

### Event Handlers
Subscribe to events for custom behavior:
```go
orchestrator.Subscribe(events.EventDeviceCompleted, myHandler)
```

## Security Considerations

1. **TLS**: All HTTP delivery uses HTTPS by default
2. **Authentication**: Delivery mechanisms handle device auth
3. **Validation**: All inputs validated before processing
4. **Timeouts**: Context timeouts prevent hanging operations

## Future Enhancements

- Scheduler component for time-based updates
- Delta/differential updates for bandwidth efficiency
- Rollback support for failed updates
- Metrics export (Prometheus)
- Web UI for monitoring
