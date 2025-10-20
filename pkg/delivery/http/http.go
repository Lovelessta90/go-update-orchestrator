package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dovaclean/go-update-orchestrator/internal/retry"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Config holds HTTP delivery configuration.
type Config struct {
	// Timeout for HTTP requests
	Timeout time.Duration

	// TLSConfig for HTTPS connections
	TLSConfig *tls.Config

	// Headers to include in all requests (e.g., Authorization)
	Headers map[string]string

	// UpdateEndpoint is the path for pushing updates (default: /update)
	UpdateEndpoint string

	// VerifyEndpoint is the path for verification (default: /version)
	VerifyEndpoint string

	// MaxRetries for transient failures
	MaxRetries int

	// RetryConfig for exponential backoff (optional, uses defaults if nil)
	RetryConfig *retry.Config

	// SkipTLSVerify bypasses certificate verification (insecure, for testing)
	SkipTLSVerify bool
}

// DefaultConfig returns sensible defaults for HTTP delivery.
func DefaultConfig() *Config {
	return &Config{
		Timeout:        30 * time.Second,
		Headers:        make(map[string]string),
		UpdateEndpoint: "/update",
		VerifyEndpoint: "/version",
		MaxRetries:     3,
		SkipTLSVerify:  false,
	}
}

// Delivery implements HTTP-based update delivery.
type Delivery struct {
	config      *Config
	client      *http.Client
	retryConfig *retry.Config
}

// New creates a new HTTP delivery mechanism with default config.
func New() *Delivery {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new HTTP delivery mechanism with custom config.
func NewWithConfig(config *Config) *Delivery {
	// Create TLS config if needed
	tlsConfig := config.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: config.SkipTLSVerify,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// Create HTTP client with configured timeout and TLS
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Setup retry configuration
	retryConfig := config.RetryConfig
	if retryConfig == nil {
		retryConfig = retry.DefaultConfig()
		retryConfig.MaxAttempts = config.MaxRetries
	}

	return &Delivery{
		config:      config,
		client:      client,
		retryConfig: retryConfig,
	}
}

// Push delivers the update payload to a device via HTTP POST.
// The payload is streamed directly to the device without loading into memory.
// Retries are supported if the payload implements io.Seeker (e.g., *os.File, *bytes.Reader).
func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	url := device.Address + d.config.UpdateEndpoint

	// Check if payload supports seeking (required for retries)
	seeker, canSeek := payload.(io.Seeker)

	// Wrap the push logic for retry
	return retry.Do(ctx, d.retryConfig, func() error {
		// Reset payload to beginning if this is a retry
		if canSeek && seeker != nil {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to reset payload for retry: %w", err)
			}
		}

		// Create HTTP request with context (supports cancellation)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set content type
		req.Header.Set("Content-Type", "application/octet-stream")

		// Add custom headers (e.g., Authorization, X-Device-ID)
		for key, value := range d.config.Headers {
			req.Header.Set(key, value)
		}

		// Add device-specific headers for tracking
		req.Header.Set("X-Device-ID", device.ID)
		req.Header.Set("X-Device-Name", device.Name)

		// Execute the request
		resp, err := d.client.Do(req)
		if err != nil {
			// Network errors are retryable
			if isRetryable(err) {
				return err // Will be retried
			}
			return fmt.Errorf("failed to push update to %s: %w", device.Address, err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Read error message from response body (limited)
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)

			// 5xx errors are retryable, 4xx are not
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				return fmt.Errorf("update push failed with status %d: %s", resp.StatusCode, string(body[:n]))
			}

			// 4xx errors should not be retried (client error)
			return &retry.NonRetryable{
				Err: fmt.Errorf("update push failed with status %d: %s", resp.StatusCode, string(body[:n])),
			}
		}

		return nil
	})
}

// isRetryable determines if an error is worth retrying
func isRetryable(err error) bool {
	// Context errors should not be retried
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are generally retryable
	return true
}

// Verify checks if the update was successfully applied via HTTP GET.
// This calls the device's version endpoint and checks the firmware version.
func (d *Delivery) Verify(ctx context.Context, device core.Device) error {
	// Build the verify URL
	url := device.Address + d.config.VerifyEndpoint

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create verify request: %w", err)
	}

	// Add custom headers
	for key, value := range d.config.Headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to verify update on %s: %w", device.Address, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify failed with status %d", resp.StatusCode)
	}

	// TODO: Parse response body and verify firmware version
	// For now, just check that the endpoint is reachable
	return nil
}
