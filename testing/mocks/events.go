package mocks

import (
	"context"
	"sync/atomic"

	"github.com/dovaclean/go-update-orchestrator/pkg/events"
)

// EventCounter returns a handler that counts events
func EventCounter(counter *atomic.Int64) events.Handler {
	return events.HandlerFunc(func(ctx context.Context, event events.Event) {
		counter.Add(1)
	})
}
