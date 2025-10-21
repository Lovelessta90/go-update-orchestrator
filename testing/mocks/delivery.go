package mocks

import (
	"context"
	"io"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// MockDelivery is a mock delivery mechanism for testing
type MockDelivery struct {
	PushCount   int
	VerifyCount int
	ShouldFail  bool
}

// NewMockDelivery creates a new mock delivery
func NewMockDelivery() *MockDelivery {
	return &MockDelivery{}
}

// Push simulates pushing an update
func (m *MockDelivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	m.PushCount++

	if m.ShouldFail {
		return core.ErrDeliveryFailed
	}

	// Drain the payload
	io.Copy(io.Discard, payload)

	return nil
}

// Verify simulates verifying an update
func (m *MockDelivery) Verify(ctx context.Context, device core.Device) error {
	m.VerifyCount++

	if m.ShouldFail {
		return core.ErrVerificationFailed
	}

	return nil
}
