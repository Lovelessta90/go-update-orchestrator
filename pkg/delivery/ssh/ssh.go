package ssh

import (
	"context"
	"io"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Delivery implements SSH-based update delivery.
type Delivery struct {
	// TODO: Add configuration fields (key auth, known_hosts, etc)
}

// New creates a new SSH delivery mechanism.
func New() *Delivery {
	return &Delivery{}
}

// Push delivers the update payload to a device via SCP/SFTP.
func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	// TODO: Implement SSH/SCP with streaming
	return nil
}

// Verify checks if the update was successfully applied via SSH command.
func (d *Delivery) Verify(ctx context.Context, device core.Device) error {
	// TODO: Implement verification (e.g., SSH command to check version)
	return nil
}
