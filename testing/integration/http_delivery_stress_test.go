package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	httpdelivery "github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
)

// TestStressLoad_1000Devices tests pushing to 1K devices concurrently
func TestStressLoad_1000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testStressLoad(t, 1000, 100) // 1K devices, 100 concurrent
}

// TestStressLoad_10000Devices tests pushing to 10K devices concurrently
func TestStressLoad_10000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testStressLoad(t, 10000, 100) // 10K devices, 100 concurrent
}

// TestStressLoad_100000Devices tests pushing to 100K devices concurrently
// This will likely find breaking points
func TestStressLoad_100000Devices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testStressLoad(t, 100000, 100) // 100K devices, 100 concurrent
}

func testStressLoad(t *testing.T, deviceCount, concurrency int) {
	t.Helper()

	var successCount, failCount int64
	var requestCount int64

	// Create server that tracks requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	start := time.Now()

	// Use worker pool to limit concurrency
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < deviceCount; i++ {
		wg.Add(1)
		go func(deviceID int) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			device := core.Device{
				ID:      fmt.Sprintf("device-%d", deviceID),
				Address: server.URL,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			payload := strings.NewReader("test payload")
			err := delivery.Push(ctx, device, payload)

			if err != nil {
				atomic.AddInt64(&failCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Report results
	t.Logf("Stress Test Results:")
	t.Logf("  Devices: %d", deviceCount)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Failed: %d", failCount)
	t.Logf("  Requests: %d", requestCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f devices/sec", float64(deviceCount)/elapsed.Seconds())

	// Check for failures
	if failCount > 0 {
		t.Errorf("Expected 0 failures, got %d", failCount)
	}

	// Sanity check: all requests should have been received
	if requestCount != int64(deviceCount) {
		t.Errorf("Expected %d requests, got %d", deviceCount, requestCount)
	}
}

// TestFailure_NetworkTimeout tests behavior when network times out
func TestFailure_NetworkTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failure test in short mode")
	}

	// Server that hangs forever
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Hour) // Hang
	}))
	defer server.Close()

	delivery := httpdelivery.New()
	device := core.Device{
		ID:      "timeout-device",
		Address: server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := delivery.Push(ctx, device, strings.NewReader("test"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("Timeout took too long: %v", elapsed)
	}

	t.Logf("Timeout error (expected): %v", err)
	t.Logf("Time to fail: %v", elapsed)
}

// TestFailure_ServerErrors tests behavior with various HTTP errors
func TestFailure_ServerErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failure test in short mode")
	}

	errorCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}

	for _, code := range errorCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(code)
			}))
			defer server.Close()

			delivery := httpdelivery.New()
			device := core.Device{
				ID:      "error-device",
				Address: server.URL,
			}

			ctx := context.Background()
			err := delivery.Push(ctx, device, strings.NewReader("test"))

			if err == nil {
				t.Fatalf("Expected error for HTTP %d, got nil", code)
			}

			t.Logf("HTTP %d error (expected): %v", code, err)
		})
	}
}

// TestFailure_RetryStorm tests behavior under retry storm conditions
func TestFailure_RetryStorm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failure test in short mode")
	}

	var requestCount int64
	failUntil := int64(5) // Fail first 5 attempts

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		io.Copy(io.Discard, r.Body)

		if count <= failUntil {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &httpdelivery.Config{
		Timeout:    5 * time.Second,
		MaxRetries: 10,
	}
	delivery := httpdelivery.NewWithConfig(config)

	device := core.Device{
		ID:      "retry-device",
		Address: server.URL,
	}

	ctx := context.Background()
	start := time.Now()
	err := delivery.Push(ctx, device, strings.NewReader("test"))
	elapsed := time.Since(start)

	t.Logf("Retry storm results:")
	t.Logf("  Requests made: %d", requestCount)
	t.Logf("  Time elapsed: %v", elapsed)
	t.Logf("  Error: %v", err)

	// Should eventually succeed after retries
	if err == nil {
		t.Logf("Request succeeded after %d attempts", requestCount)
	}
}

// TestFailure_ConnectionRefused tests behavior when connection is refused
func TestFailure_ConnectionRefused(t *testing.T) {
	delivery := httpdelivery.New()
	device := core.Device{
		ID:      "unreachable-device",
		Address: "http://localhost:59999", // Unlikely to be in use
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := delivery.Push(ctx, device, strings.NewReader("test"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected connection refused error, got nil")
	}

	t.Logf("Connection refused error (expected): %v", err)
	t.Logf("Time to fail: %v", elapsed)
}

// TestResourceExhaustion_GoroutineLeak tests for goroutine leaks under load
func TestResourceExhaustion_GoroutineLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource exhaustion test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	// Measure goroutines before
	beforeGoroutines := countGoroutines()

	// Make many requests
	iterations := 1000
	for i := 0; i < iterations; i++ {
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: server.URL,
		}

		ctx := context.Background()
		err := delivery.Push(ctx, device, strings.NewReader("test"))
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}
	}

	// Force garbage collection
	time.Sleep(100 * time.Millisecond)

	// Measure goroutines after
	afterGoroutines := countGoroutines()

	t.Logf("Goroutine leak test:")
	t.Logf("  Before: %d goroutines", beforeGoroutines)
	t.Logf("  After:  %d goroutines", afterGoroutines)
	t.Logf("  Delta:  %d goroutines", afterGoroutines-beforeGoroutines)

	// Allow some variance (HTTP client may keep connections alive)
	// but shouldn't grow linearly with request count
	maxAllowedGrowth := 50
	if afterGoroutines-beforeGoroutines > maxAllowedGrowth {
		t.Errorf("Potential goroutine leak: grew by %d goroutines (max allowed: %d)",
			afterGoroutines-beforeGoroutines, maxAllowedGrowth)
	}
}

// TestResourceExhaustion_ConnectionPool tests connection pool limits
func TestResourceExhaustion_ConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource exhaustion test in short mode")
	}

	var activeConnections int64
	var maxConnections int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&activeConnections, 1)
		defer atomic.AddInt64(&activeConnections, -1)

		// Track max
		for {
			max := atomic.LoadInt64(&maxConnections)
			if current <= max || atomic.CompareAndSwapInt64(&maxConnections, max, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond) // Simulate work
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	// Fire 100 concurrent requests
	concurrency := 100
	var wg sync.WaitGroup

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			device := core.Device{
				ID:      fmt.Sprintf("device-%d", id),
				Address: server.URL,
			}

			ctx := context.Background()
			err := delivery.Push(ctx, device, strings.NewReader("test"))
			if err != nil {
				t.Errorf("Push failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Connection pool test:")
	t.Logf("  Concurrent requests: %d", concurrency)
	t.Logf("  Max simultaneous connections: %d", maxConnections)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", float64(concurrency)/elapsed.Seconds())
}

// TestResourceExhaustion_MemoryPressure tests memory usage with large payloads
func TestResourceExhaustion_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource exhaustion test in short mode")
	}

	var bytesReceived int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		atomic.AddInt64(&bytesReceived, n)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	// Send 100 concurrent 10MB payloads (1GB total)
	payloadSize := 10 * 1024 * 1024 // 10 MB
	concurrency := 100

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			device := core.Device{
				ID:      fmt.Sprintf("device-%d", id),
				Address: server.URL,
			}

			// Create large payload
			payload := strings.NewReader(strings.Repeat("A", payloadSize))

			ctx := context.Background()
			err := delivery.Push(ctx, device, payload)
			if err != nil {
				t.Errorf("Push failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalGB := float64(bytesReceived) / (1024 * 1024 * 1024)

	t.Logf("Memory pressure test:")
	t.Logf("  Concurrent uploads: %d", concurrency)
	t.Logf("  Payload size: %d MB", payloadSize/(1024*1024))
	t.Logf("  Total data: %.2f GB", totalGB)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f MB/sec", totalGB*1024/elapsed.Seconds())
}

// TestChaos_RandomFailures tests behavior with random failures
func TestChaos_RandomFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	var successCount, failCount int64
	failureRate := 0.3 // 30% failure rate

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)

		// Randomly fail 30% of requests
		if float64(time.Now().UnixNano()%100)/100 < failureRate {
			w.WriteHeader(http.StatusServiceUnavailable)
			atomic.AddInt64(&failCount, 1)
		} else {
			w.WriteHeader(http.StatusOK)
			atomic.AddInt64(&successCount, 1)
		}
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	// Make 100 requests
	totalRequests := 100
	for i := 0; i < totalRequests; i++ {
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: server.URL,
		}

		ctx := context.Background()
		_ = delivery.Push(ctx, device, strings.NewReader("test"))
	}

	t.Logf("Chaos test (random failures):")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Server successes: %d", successCount)
	t.Logf("  Server failures: %d", failCount)
	t.Logf("  Expected failure rate: %.1f%%", failureRate*100)
	t.Logf("  Actual failure rate: %.1f%%", float64(failCount)/float64(totalRequests)*100)
}

// TestChaos_SlowNetwork tests behavior with slow network
func TestChaos_SlowNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Random latency between 100-500ms
		latency := time.Duration(100+time.Now().UnixNano()%400) * time.Millisecond
		time.Sleep(latency)

		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()

	totalRequests := 10
	var totalDuration time.Duration

	for i := 0; i < totalRequests; i++ {
		device := core.Device{
			ID:      fmt.Sprintf("device-%d", i),
			Address: server.URL,
		}

		ctx := context.Background()
		start := time.Now()
		err := delivery.Push(ctx, device, strings.NewReader("test"))
		elapsed := time.Since(start)

		totalDuration += elapsed

		if err != nil {
			t.Errorf("Push failed: %v", err)
		}
	}

	avgDuration := totalDuration / time.Duration(totalRequests)

	t.Logf("Chaos test (slow network):")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Average latency: %v", avgDuration)
	t.Logf("  Total time: %v", totalDuration)
}

// TestEdgeCase_ZeroBytePayload tests sending zero bytes
func TestEdgeCase_ZeroBytePayload(t *testing.T) {
	var receivedBytes int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		atomic.AddInt64(&receivedBytes, n)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()
	device := core.Device{
		ID:      "zero-byte-device",
		Address: server.URL,
	}

	ctx := context.Background()
	err := delivery.Push(ctx, device, strings.NewReader("")) // Zero bytes

	if err != nil {
		t.Fatalf("Zero-byte push failed: %v", err)
	}

	if receivedBytes != 0 {
		t.Errorf("Expected 0 bytes received, got %d", receivedBytes)
	}

	t.Logf("Zero-byte payload: success (received %d bytes)", receivedBytes)
}

// TestEdgeCase_HugePayload tests sending extremely large payload
func TestEdgeCase_HugePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case test in short mode")
	}

	var receivedBytes int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		atomic.AddInt64(&receivedBytes, n)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	delivery := httpdelivery.New()
	device := core.Device{
		ID:      "huge-payload-device",
		Address: server.URL,
	}

	// 100 MB payload
	payloadSize := 100 * 1024 * 1024
	payload := strings.NewReader(strings.Repeat("A", payloadSize))

	ctx := context.Background()
	start := time.Now()
	err := delivery.Push(ctx, device, payload)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Huge payload push failed: %v", err)
	}

	if receivedBytes != int64(payloadSize) {
		t.Errorf("Expected %d bytes received, got %d", payloadSize, receivedBytes)
	}

	t.Logf("Huge payload test:")
	t.Logf("  Payload size: %d MB", payloadSize/(1024*1024))
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f MB/sec", float64(payloadSize)/(1024*1024)/elapsed.Seconds())
}

// TestEdgeCase_MalformedResponse tests handling of malformed HTTP responses
func TestEdgeCase_MalformedResponse(t *testing.T) {
	// Create a raw TCP server that sends garbage
	listener, err := testNewListener()
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send garbage
		conn.Write([]byte("NOT HTTP RESPONSE\r\n\r\n"))
	}()

	delivery := httpdelivery.New()
	device := core.Device{
		ID:      "malformed-device",
		Address: "http://" + listener.Addr().String(),
	}

	ctx := context.Background()
	err = delivery.Push(ctx, device, strings.NewReader("test"))

	if err == nil {
		t.Fatal("Expected error from malformed response, got nil")
	}

	t.Logf("Malformed response error (expected): %v", err)
}

// Helper function to count goroutines
func countGoroutines() int {
	// Give some time for cleanup
	time.Sleep(50 * time.Millisecond)
	return runtime.NumGoroutine()
}

// Helper to create TCP listener
func testNewListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
