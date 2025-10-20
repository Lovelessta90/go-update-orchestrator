package orchestrator

import "errors"

// Config holds orchestrator configuration.
type Config struct {
	// MaxConcurrent is the maximum number of concurrent device updates.
	MaxConcurrent int

	// RetryAttempts is the number of retry attempts for failed updates.
	RetryAttempts int

	// EventBufferSize is the buffer size for the event bus.
	EventBufferSize int

	// PayloadBufferSize is the buffer size for streaming payloads (bytes).
	PayloadBufferSize int
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrent:     100,
		RetryAttempts:     3,
		EventBufferSize:   1000,
		PayloadBufferSize: 1024 * 1024, // 1MB
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.MaxConcurrent < 1 {
		return errors.New("MaxConcurrent must be at least 1")
	}
	if c.RetryAttempts < 0 {
		return errors.New("RetryAttempts cannot be negative")
	}
	if c.EventBufferSize < 0 {
		return errors.New("EventBufferSize cannot be negative")
	}
	if c.PayloadBufferSize < 1024 {
		return errors.New("PayloadBufferSize must be at least 1024 bytes")
	}
	return nil
}
