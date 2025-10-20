# Go Update Orchestrator

**Efficiently push software/firmware updates to thousands of devices concurrently with proper orchestration, progress tracking, and failure handling.**

## Overview

Go Update Orchestrator is a lightweight, high-performance tool for managing large-scale device updates. Built for companies managing fleets of devices (POS systems, IoT devices, digital signage, vehicles, medical equipment), it replaces vendor-locked tools and brittle bash scripts with a robust, portable solution.

## Key Features

- **Massive Scale** - Handle 10,000+ devices efficiently with bounded concurrency
- **Protocol Agnostic** - Pluggable delivery mechanisms (HTTP, SSH, custom)
- **Streaming Updates** - Memory-efficient streaming, no full payload loading
- **Progress Tracking** - Real-time progress monitoring with accurate time estimates
- **Failure Recovery** - Automatic retry with exponential backoff, resume failed updates
- **Single Binary** - Zero dependencies, no CGO, maximum portability
- **Event-Driven** - Components communicate via events, not direct coupling
- **Context-Based Cancellation** - Graceful shutdown throughout

## Architecture

### Core Components

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Scheduler  │────▶│ Orchestrator │◀────│  Registry   │
└─────────────┘     └──────────────┘     └─────────────┘
                            │
                    ┌───────┴───────┐
                    ▼               ▼
            ┌──────────────┐  ┌──────────┐
            │   Delivery   │  │  Events  │
            └──────────────┘  └──────────┘
                    │               │
                    └───────┬───────┘
                            ▼
                    ┌──────────────┐
                    │   Progress   │
                    └──────────────┘
```

- **Scheduler** - Manages when/how updates happen
- **Orchestrator** - Coordinates all components
- **Registry** - Stores device information
- **Delivery** - Protocol-agnostic push system
- **Events** - Event bus for component communication
- **Progress** - Tracks and estimates completion

### Design Principles

- **Interface-based** - All major components use interfaces
- **Single Responsibility** - Each component has one job
- **Zero Core Dependencies** - Pure Go stdlib for core
- **Streaming** - Don't load full payloads in memory
- **Bounded Concurrency** - Worker pools prevent resource exhaustion
- **Context Everywhere** - Proper cancellation throughout

## Quick Start

### Installation

```bash
go install github.com/dovaclean/go-update-orchestrator/cmd/orchestrator@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
    "github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
    "github.com/dovaclean/go-update-orchestrator/pkg/registry/memory"
)

func main() {
    // Create components
    reg := memory.New()
    delivery := http.New()

    // Configure orchestrator
    config := &orchestrator.Config{
        MaxConcurrent: 100,
        RetryAttempts: 3,
    }

    orch, err := orchestrator.New(config, reg, delivery)
    if err != nil {
        log.Fatal(err)
    }

    // Execute update
    ctx := context.Background()
    if err := orch.ExecuteUpdate(ctx, updateJob); err != nil {
        log.Fatal(err)
    }
}
```

See [examples/](examples/) for complete working examples.

## Use Cases

### POS Systems
Update thousands of point-of-sale terminals overnight with new software versions.

### IoT Devices
Push firmware updates to distributed sensor networks, edge devices, or smart home systems.

### Digital Signage
Deploy content and configuration updates to displays across multiple locations.

### Vehicle Fleets
Update telematics units, infotainment systems, or diagnostic equipment.

## Performance

- **10,000+ devices** - Efficiently handles massive fleets
- **Streaming updates** - Constant memory usage regardless of payload size
- **Bounded concurrency** - Configurable worker pools prevent resource exhaustion
- **Zero allocations** - Hot path optimized for minimal GC pressure

## Project Structure

```
go-update-orchestrator/
├── cmd/                    # Command-line tools
│   ├── orchestrator/      # CLI tool
│   └── server/            # HTTP API server
├── pkg/                   # Public API
│   ├── core/             # Core types and interfaces
│   ├── delivery/         # Delivery mechanisms
│   ├── registry/         # Device registries
│   ├── events/           # Event system
│   ├── progress/         # Progress tracking
│   └── orchestrator/     # Main orchestrator
├── internal/             # Private implementation
├── testing/              # Test doubles and integration tests
├── examples/             # Usage examples
└── docs/                 # Documentation
```

## Development

### Build

```bash
make build          # Build all binaries
make install        # Install to GOPATH/bin
```

### Test

```bash
make test                # Run all tests
make test-unit           # Unit tests only
make test-integration    # Integration tests only
make bench               # Benchmarks
```

### Code Quality

```bash
make fmt           # Format code
make vet           # Run go vet
make lint          # Run linters
```

## Documentation

- [Architecture](docs/architecture.md) - Design decisions and patterns
- [Interfaces](docs/interfaces.md) - Interface contracts
- [Performance](docs/performance.md) - Performance tuning guide
- [Examples](docs/examples.md) - Usage patterns

## Similar Tools

- **Fleet/Puppet/Ansible** - Configuration management, not optimized for binary updates
- **Vendor Tools** - Often proprietary and locked to specific hardware

Go Update Orchestrator fills the gap for actively pushing binary updates at scale.

## Contributing

Contributions welcome! Please open an issue first to discuss proposed changes.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Roadmap

- [ ] HTTP delivery implementation
- [ ] SQLite registry implementation
- [ ] SSH delivery mechanism
- [ ] Web UI for progress monitoring
- [ ] Prometheus metrics export
- [ ] Delta/differential updates
- [ ] Rollback support
- [ ] Device grouping/tagging

## Support

- GitHub Issues: Report bugs or request features
- Documentation: Check [docs/](docs/) directory
- Examples: See [examples/](examples/) for working code
