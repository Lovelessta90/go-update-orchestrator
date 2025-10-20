package integration

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
	httpdelivery "github.com/dovaclean/go-update-orchestrator/pkg/delivery/http"
)

// BenchmarkRealisticScenario simulates a more realistic network scenario
func BenchmarkRealisticScenario(b *testing.B) {
	scenarios := []struct {
		name    string
		latency time.Duration
		size    int
	}{
		{"LAN_1ms_1KB", 1 * time.Millisecond, 1024},
		{"LAN_1ms_1MB", 1 * time.Millisecond, 1024 * 1024},
		{"Internet_50ms_1KB", 50 * time.Millisecond, 1024},
		{"Internet_50ms_1MB", 50 * time.Millisecond, 1024 * 1024},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create server with artificial latency
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate network latency
				time.Sleep(scenario.latency)

				// Read and discard body
				io.Copy(io.Discard, r.Body)

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			delivery := httpdelivery.New()
			device := core.Device{
				ID:      "benchmark-device",
				Address: server.URL,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				payload := strings.NewReader(strings.Repeat("A", scenario.size))

				err := delivery.Push(ctx, device, payload)
				if err != nil {
					b.Fatalf("Push failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConcurrentLoad tests behavior under concurrent load
func BenchmarkConcurrentLoad(b *testing.B) {
	concurrencies := []int{1, 10, 100}

	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("Concurrent_%d", concurrency), func(b *testing.B) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate 5ms processing time
				time.Sleep(5 * time.Millisecond)
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			delivery := httpdelivery.New()
			device := core.Device{
				ID:      "benchmark-device",
				Address: server.URL,
			}

			b.ResetTimer()
			b.SetParallelism(concurrency)

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					ctx := context.Background()
					payload := strings.NewReader("test payload")

					err := delivery.Push(ctx, device, payload)
					if err != nil {
						b.Fatalf("Push failed: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkMemoryPressure tests behavior under memory constraints
func BenchmarkMemoryPressure(b *testing.B) {
	sizes := []int{
		1024,           // 1 KB
		1024 * 1024,    // 1 MB
		10 * 1024 * 1024, // 10 MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Payload_%dMB", size/(1024*1024)), func(b *testing.B) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			delivery := httpdelivery.New()
			device := core.Device{
				ID:      "benchmark-device",
				Address: server.URL,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				payload := strings.NewReader(strings.Repeat("A", size))

				err := delivery.Push(ctx, device, payload)
				if err != nil {
					b.Fatalf("Push failed: %v", err)
				}
			}

			// Report throughput
			b.SetBytes(int64(size))
		})
	}
}
