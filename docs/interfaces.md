# Interface Contracts

## Core Interfaces

### Delivery Interface

```go
type Delivery interface {
    Push(ctx context.Context, device core.Device, payload io.Reader) error
    Verify(ctx context.Context, device core.Device) error
}
```

**Purpose**: Protocol-agnostic update delivery to devices.

**Contracts**:
- `Push()` must stream `payload` without loading into memory
- `Push()` must respect context cancellation
- `Push()` must be idempotent (safe to retry)
- `Verify()` confirms successful update application
- Implementations must handle their own authentication

**Error Handling**:
- Return `core.ErrDeliveryFailed` for delivery failures
- Wrap underlying errors for context

---

### Registry Interface

```go
type Registry interface {
    List(ctx context.Context, filter core.Filter) ([]core.Device, error)
    Get(ctx context.Context, id string) (*core.Device, error)
    Add(ctx context.Context, device core.Device) error
    Update(ctx context.Context, device core.Device) error
    Delete(ctx context.Context, id string) error
}
```

**Purpose**: Device storage and retrieval.

**Contracts**:
- `List()` must support filtering by tags and IDs
- `Get()` returns `core.ErrDeviceNotFound` if device doesn't exist
- `Add()` must validate device before storing
- `Update()` returns `core.ErrDeviceNotFound` if device doesn't exist
- All operations must be thread-safe

**Performance**:
- `Get()` should be O(1) or O(log n)
- `List()` should support pagination via Filter.Offset/Limit

---

### Scheduler Interface

```go
type Scheduler interface {
    Schedule(ctx context.Context, update Update) error
    Status(ctx context.Context, updateID string) (*Status, error)
    Cancel(ctx context.Context, updateID string) error
    List(ctx context.Context, status UpdateStatus) ([]Status, error)
}
```

**Purpose**: Manage update scheduling and execution timing.

**Contracts**:
- `Schedule()` queues update for execution
- `Status()` returns current update state
- `Cancel()` attempts graceful cancellation
- `List()` returns all updates with given status

---

### Progress Tracker Interface

```go
type Tracker interface {
    Start(ctx context.Context, updateID string, totalDevices int)
    UpdateDevice(ctx context.Context, updateID, deviceID string, status string, bytesTransferred int64)
    Complete(ctx context.Context, updateID string)
    GetProgress(ctx context.Context, updateID string) (*Progress, error)
}
```

**Purpose**: Track update progress and estimate completion time.

**Contracts**:
- `Start()` initializes tracking for an update
- `UpdateDevice()` records per-device progress
- `Complete()` marks update as finished
- `GetProgress()` returns current progress snapshot
- Must be thread-safe for concurrent updates

---

### Event Handler Interface

```go
type Handler interface {
    Handle(ctx context.Context, event Event)
}
```

**Purpose**: Process events from the event bus.

**Contracts**:
- `Handle()` must be non-blocking
- `Handle()` must handle errors internally
- Multiple handlers can process the same event

---

## Implementation Guidelines

### Thread Safety
All interface implementations must be thread-safe. Use:
- `sync.Mutex` for exclusive access
- `sync.RWMutex` for read-heavy workloads
- Atomic operations for counters

### Context Handling
- Always check `ctx.Done()` in loops
- Pass context to downstream calls
- Return `ctx.Err()` on cancellation

### Error Wrapping
Use `fmt.Errorf()` with `%w` to wrap errors:
```go
return fmt.Errorf("failed to push to device %s: %w", device.ID, err)
```

### Resource Cleanup
- Use `defer` for cleanup
- Close readers/writers
- Cancel child contexts

## Testing Interface Implementations

### Compliance Tests
Create interface compliance tests:
```go
func TestDeliveryCompliance(t *testing.T, d delivery.Delivery) {
    // Test all interface methods
    // Verify contracts are honored
}
```

### Mock Implementations
Provide mocks in `testing/mocks/`:
```go
type MockDelivery struct {
    PushFunc   func(ctx context.Context, device core.Device, payload io.Reader) error
    VerifyFunc func(ctx context.Context, device core.Device) error
}
```

## Versioning

Interfaces follow semantic versioning:
- **Major**: Breaking changes to interface signatures
- **Minor**: New methods added (with default implementations)
- **Patch**: Documentation/implementation improvements
