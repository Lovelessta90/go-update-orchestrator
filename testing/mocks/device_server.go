package mocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// DeviceServer simulates a POS device HTTP server for testing.
type DeviceServer struct {
	server          *httptest.Server
	mu              sync.RWMutex
	firmwareVersion string
	updateCount     int
	lastUpdateTime  time.Time
	lastUpdateSize  int64
	failNext        bool // Simulate failure on next update
	alwaysFail      bool // Always fail all updates
}

// VersionResponse is the JSON response from /version endpoint.
type VersionResponse struct {
	Version    string    `json:"version"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
}

// UpdateResponse is the JSON response from /update endpoint.
type UpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Size    int64  `json:"size"`
}

// NewDeviceServer creates a new mock device server.
func NewDeviceServer(initialVersion string) *DeviceServer {
	ds := &DeviceServer{
		firmwareVersion: initialVersion,
	}

	mux := http.NewServeMux()

	// POST /update - Receive firmware update
	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ds.mu.Lock()
		defer ds.mu.Unlock()

		// Simulate permanent failure if configured
		if ds.alwaysFail {
			http.Error(w, "Simulated update failure", http.StatusInternalServerError)
			return
		}

		// Simulate failure if configured (one-time)
		if ds.failNext {
			ds.failNext = false
			http.Error(w, "Simulated update failure", http.StatusInternalServerError)
			return
		}

		// Read the update payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Simulate update process
		ds.lastUpdateSize = int64(len(body))
		ds.lastUpdateTime = time.Now()
		ds.updateCount++

		// Parse firmware version from payload (simple: first 10 bytes)
		if len(body) > 10 {
			ds.firmwareVersion = string(body[:10])
		}

		// Send success response
		response := UpdateResponse{
			Success: true,
			Message: "Update installed successfully",
			Size:    ds.lastUpdateSize,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// GET /version - Get current firmware version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ds.mu.RLock()
		defer ds.mu.RUnlock()

		response := VersionResponse{
			Version:    ds.firmwareVersion,
			UpdatedAt:  ds.lastUpdateTime,
			DeviceID:   r.Header.Get("X-Device-ID"),
			DeviceName: r.Header.Get("X-Device-Name"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// GET /health - Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create test server
	ds.server = httptest.NewServer(mux)

	return ds
}

// URL returns the server URL.
func (ds *DeviceServer) URL() string {
	return ds.server.URL
}

// Close shuts down the server.
func (ds *DeviceServer) Close() {
	ds.server.Close()
}

// GetFirmwareVersion returns the current firmware version.
func (ds *DeviceServer) GetFirmwareVersion() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.firmwareVersion
}

// GetUpdateCount returns the number of updates received.
func (ds *DeviceServer) GetUpdateCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.updateCount
}

// GetLastUpdateSize returns the size of the last update.
func (ds *DeviceServer) GetLastUpdateSize() int64 {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.lastUpdateSize
}

// SetFailNext configures the server to fail the next update.
func (ds *DeviceServer) SetFailNext(fail bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.failNext = fail
}

// SetAlwaysFail configures the server to always fail updates.
func (ds *DeviceServer) SetAlwaysFail(fail bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.alwaysFail = fail
}

// Reset resets the server state.
func (ds *DeviceServer) Reset() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.updateCount = 0
	ds.lastUpdateSize = 0
	ds.lastUpdateTime = time.Time{}
	ds.failNext = false
	ds.alwaysFail = false
}

// MultiDeviceServer manages multiple mock device servers.
type MultiDeviceServer struct {
	servers map[string]*DeviceServer
	mu      sync.RWMutex
}

// NewMultiDeviceServer creates a manager for multiple device servers.
func NewMultiDeviceServer() *MultiDeviceServer {
	return &MultiDeviceServer{
		servers: make(map[string]*DeviceServer),
	}
}

// AddDevice creates a new mock device server.
func (m *MultiDeviceServer) AddDevice(deviceID, initialVersion string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	server := NewDeviceServer(initialVersion)
	m.servers[deviceID] = server
	return server.URL()
}

// GetDevice returns a device server by ID.
func (m *MultiDeviceServer) GetDevice(deviceID string) (*DeviceServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[deviceID]
	if !ok {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}
	return server, nil
}

// CloseAll shuts down all device servers.
func (m *MultiDeviceServer) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, server := range m.servers {
		server.Close()
	}
	m.servers = make(map[string]*DeviceServer)
}

// GetAllUpdateCounts returns update counts for all devices.
func (m *MultiDeviceServer) GetAllUpdateCounts() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]int)
	for id, server := range m.servers {
		counts[id] = server.GetUpdateCount()
	}
	return counts
}
