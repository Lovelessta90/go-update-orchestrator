package retry

import (
	"context"
	"math"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxAttempts  int           // Maximum number of retry attempts
	InitialDelay time.Duration // Initial backoff delay
	MaxDelay     time.Duration // Maximum backoff delay
	Multiplier   float64       // Backoff multiplier
}

// DefaultConfig returns a retry configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// Do executes a function with exponential backoff retry logic.
func Do(ctx context.Context, config *Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Try to execute the function
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate backoff delay
		delay := calculateBackoff(config, attempt)

		// Wait with context cancellation support
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// calculateBackoff calculates the backoff delay for a given attempt.
func calculateBackoff(config *Config, attempt int) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	if delay > float64(config.MaxDelay) {
		return config.MaxDelay
	}

	return time.Duration(delay)
}
