package http

import (
	"context"
	"io"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Delivery implements HTTP-based update delivery.
type Delivery struct {
	// TODO: Add configuration fields (timeout, TLS, auth, etc)
}

// New creates a new HTTP delivery mechanism.
func New() *Delivery {
	return &Delivery{}
}

// Push delivers the update payload to a device via HTTP POST.
func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	// TODO: Implement HTTP POST with streaming
	return nil
}

// Verify checks if the update was successfully applied via HTTP GET.
func (d *Delivery) Verify(ctx context.Context, device core.Device) error {
	// TODO: Implement verification (e.g., GET /version endpoint)
	return nil
}
