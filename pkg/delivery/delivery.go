package delivery

import (
	"context"
	"io"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Delivery defines the interface for pushing updates to devices.
// Implementations handle protocol-specific delivery (HTTP, SSH, etc).
type Delivery interface {
	// Push delivers the update payload to the specified device.
	// The payload reader should be streamed to avoid loading into memory.
	Push(ctx context.Context, device core.Device, payload io.Reader) error

	// Verify checks if the update was successfully applied on the device.
	// This can include checksum verification, version checks, etc.
	Verify(ctx context.Context, device core.Device) error
}
