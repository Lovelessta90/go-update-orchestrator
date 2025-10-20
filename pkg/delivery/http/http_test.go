package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

func TestNew(t *testing.T) {
	delivery := New()
	if delivery == nil {
		t.Fatal("New() returned nil")
	}
	if delivery.config == nil {
		t.Fatal("delivery.config is nil")
	}
	if delivery.client == nil {
		t.Fatal("delivery.client is nil")
	}
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		Timeout:        5 * time.Second,
		UpdateEndpoint: "/custom-update",
		VerifyEndpoint: "/custom-version",
		Headers:        map[string]string{"X-Custom": "value"},
	}

	delivery := NewWithConfig(config)
	if delivery == nil {
		t.Fatal("NewWithConfig() returned nil")
	}
	if delivery.config.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", delivery.config.Timeout)
	}
	if delivery.config.UpdateEndpoint != "/custom-update" {
		t.Errorf("expected update endpoint /custom-update, got %s", delivery.config.UpdateEndpoint)
	}
}

func TestPush_Success(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify endpoint
		if r.URL.Path != "/update" {
			t.Errorf("expected /update, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("expected Content-Type application/octet-stream, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Device-ID") != "test-device-001" {
			t.Errorf("expected X-Device-ID test-device-001, got %s", r.Header.Get("X-Device-ID"))
		}

		// Read and verify payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if string(body) != "test payload data" {
			t.Errorf("expected 'test payload data', got %s", string(body))
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Update received"))
	}))
	defer server.Close()

	// Create delivery
	delivery := New()

	// Create test device
	device := core.Device{
		ID:      "test-device-001",
		Name:    "Test Device",
		Address: server.URL,
	}

	// Create test payload
	payload := strings.NewReader("test payload data")

	// Execute push
	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}
}

func TestPush_Failure_BadStatus(t *testing.T) {
	// Create mock HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}
	payload := strings.NewReader("test payload")

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestPush_Failure_NetworkError(t *testing.T) {
	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: "http://localhost:99999", // Invalid port
	}
	payload := strings.NewReader("test payload")

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPush_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}
	payload := strings.NewReader("test payload")

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestPush_CustomHeaders(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create delivery with custom headers
	config := DefaultConfig()
	config.Headers = map[string]string{
		"Authorization": "Bearer test-token",
		"X-Custom":      "value",
	}
	delivery := NewWithConfig(config)

	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}
	payload := strings.NewReader("test payload")

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}
}

func TestPush_Streaming(t *testing.T) {
	// Create a large payload to test streaming
	largePayload := strings.Repeat("A", 1024*1024) // 1MB

	receivedSize := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body in chunks to verify streaming
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			receivedSize += n
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading body: %v", err)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}
	payload := strings.NewReader(largePayload)

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push() failed: %v", err)
	}

	if receivedSize != len(largePayload) {
		t.Errorf("expected %d bytes, received %d bytes", len(largePayload), receivedSize)
	}
}

func TestVerify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Verify endpoint
		if r.URL.Path != "/version" {
			t.Errorf("expected /version, got %s", r.URL.Path)
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "2.4.0"}`))
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}

	ctx := context.Background()
	err := delivery.Verify(ctx, device)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}
}

func TestVerify_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}

	ctx := context.Background()
	err := delivery.Verify(ctx, device)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVerify_CustomEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Errorf("expected /api/status, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.VerifyEndpoint = "/api/status"
	delivery := NewWithConfig(config)

	device := core.Device{
		ID:      "test-device-001",
		Address: server.URL,
	}

	ctx := context.Background()
	err := delivery.Verify(ctx, device)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}
	if config.UpdateEndpoint != "/update" {
		t.Errorf("expected update endpoint /update, got %s", config.UpdateEndpoint)
	}
	if config.VerifyEndpoint != "/version" {
		t.Errorf("expected verify endpoint /version, got %s", config.VerifyEndpoint)
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}
	if config.SkipTLSVerify {
		t.Error("expected SkipTLSVerify false by default")
	}
}

// Benchmark tests
func BenchmarkPush(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "bench-device",
		Address: server.URL,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := strings.NewReader("benchmark payload data")
		err := delivery.Push(ctx, device, payload)
		if err != nil {
			b.Fatalf("Push() failed: %v", err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "2.4.0"}`))
	}))
	defer server.Close()

	delivery := New()
	device := core.Device{
		ID:      "bench-device",
		Address: server.URL,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := delivery.Verify(ctx, device)
		if err != nil {
			b.Fatalf("Verify() failed: %v", err)
		}
	}
}

// Example tests
func ExampleNew() {
	delivery := New()
	device := core.Device{
		ID:      "pos-001",
		Address: "https://pos-001.example.com:8443",
	}

	ctx := context.Background()
	payload := strings.NewReader("firmware v2.4.0")

	err := delivery.Push(ctx, device, payload)
	if err != nil {
		fmt.Printf("failed to push: %v\n", err)
	}
}

func ExampleNewWithConfig() {
	config := &Config{
		Timeout:        10 * time.Second,
		UpdateEndpoint: "/api/firmware",
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
		},
	}

	delivery := NewWithConfig(config)
	device := core.Device{
		ID:      "pos-001",
		Address: "https://pos-001.example.com:8443",
	}

	ctx := context.Background()
	err := delivery.Verify(ctx, device)
	if err != nil {
		fmt.Printf("verification failed: %v\n", err)
	}
}
