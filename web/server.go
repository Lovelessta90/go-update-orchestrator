package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry"
	"github.com/dovaclean/go-update-orchestrator/pkg/scheduler"
)

// Server is the web UI server.
type Server struct {
	addr         string
	orchestrator *orchestrator.Orchestrator
	scheduler    *scheduler.Scheduler
	registry     registry.Registry

	// WebSocket management
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex

	templates *template.Template
}

// Config holds web server configuration.
type Config struct {
	Address string // Server address (e.g., ":8080")
}

// DefaultConfig returns default web server configuration.
func DefaultConfig() *Config {
	return &Config{
		Address: ":8080",
	}
}

// New creates a new web server.
func New(config *Config, orch *orchestrator.Orchestrator, sched *scheduler.Scheduler, reg registry.Registry) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Parse templates
	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		// Templates might not exist yet, create a basic one in memory
		templates = template.New("index")
	}

	s := &Server{
		addr:         config.Address,
		orchestrator: orch,
		scheduler:    sched,
		registry:     reg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:   make(map[*websocket.Conn]bool),
		templates: templates,
	}

	return s, nil
}

// Start starts the web server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Pages
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/devices", s.handleDevicesPage)
	mux.HandleFunc("/updates", s.handleUpdatesPage)

	// API endpoints
	mux.HandleFunc("/api/devices", s.handleDevicesAPI)
	mux.HandleFunc("/api/devices/", s.handleDeviceAPI)
	mux.HandleFunc("/api/updates", s.handleUpdatesAPI)
	mux.HandleFunc("/api/updates/", s.handleUpdateAPI)
	mux.HandleFunc("/api/updates/schedule", s.handleScheduleUpdate)
	mux.HandleFunc("/api/updates/cancel", s.handleCancelUpdate)

	// WebSocket for real-time updates
	mux.HandleFunc("/ws", s.handleWebSocket)

	log.Printf("Web UI starting on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// Page Handlers

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Update Orchestrator</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        h1 { color: #333; }
        .nav { margin: 20px 0; }
        .nav a { margin-right: 15px; color: #007bff; text-decoration: none; }
        .nav a:hover { text-decoration: underline; }
        .stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin: 20px 0; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 5px; text-align: center; }
        .stat-card h3 { margin: 0; color: #666; font-size: 14px; }
        .stat-card .value { font-size: 36px; color: #007bff; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Update Orchestrator</h1>
        <div class="nav">
            <a href="/">Dashboard</a>
            <a href="/devices">Devices</a>
            <a href="/updates">Updates</a>
        </div>
        <div class="stats">
            <div class="stat-card">
                <h3>TOTAL DEVICES</h3>
                <div class="value" id="total-devices">-</div>
            </div>
            <div class="stat-card">
                <h3>ONLINE DEVICES</h3>
                <div class="value" id="online-devices">-</div>
            </div>
            <div class="stat-card">
                <h3>ACTIVE UPDATES</h3>
                <div class="value" id="active-updates">-</div>
            </div>
        </div>
        <h2>Recent Updates</h2>
        <div id="recent-updates">Loading...</div>
    </div>
    <script>
        // Fetch dashboard stats
        async function loadStats() {
            try {
                const devicesResp = await fetch('/api/devices');
                const devices = await devicesResp.json();
                document.getElementById('total-devices').textContent = devices.length;
                const online = devices.filter(d => d.status === 'online').length;
                document.getElementById('online-devices').textContent = online;

                const updatesResp = await fetch('/api/updates');
                const updates = await updatesResp.json();
                const active = updates.filter(u => u.status === 'in_progress').length;
                document.getElementById('active-updates').textContent = active;

                // Show recent updates
                const recentHTML = updates.slice(0, 5).map(u =>
                    '<div style="padding: 10px; border-bottom: 1px solid #eee;">' +
                    '<strong>' + u.id + '</strong> - ' + u.status +
                    '</div>'
                ).join('');
                document.getElementById('recent-updates').innerHTML = recentHTML || 'No updates';
            } catch (err) {
                console.error('Failed to load stats:', err);
            }
        }
        loadStats();
        setInterval(loadStats, 5000); // Refresh every 5 seconds
    </script>
</body>
</html>
`)
}

func (s *Server) handleDevicesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Devices - Update Orchestrator</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        h1 { color: #333; }
        .nav { margin: 20px 0; }
        .nav a { margin-right: 15px; color: #007bff; text-decoration: none; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; font-weight: 600; }
        .status-online { color: #28a745; }
        .status-offline { color: #dc3545; }
        .status-unknown { color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📱 Devices</h1>
        <div class="nav">
            <a href="/">Dashboard</a>
            <a href="/devices">Devices</a>
            <a href="/updates">Updates</a>
        </div>
        <table id="devices-table">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Name</th>
                    <th>Address</th>
                    <th>Status</th>
                    <th>Firmware</th>
                    <th>Location</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>
    <script>
        async function loadDevices() {
            try {
                const resp = await fetch('/api/devices');
                const devices = await resp.json();
                const tbody = document.querySelector('#devices-table tbody');
                tbody.innerHTML = devices.map(d =>
                    '<tr>' +
                    '<td>' + d.id + '</td>' +
                    '<td>' + (d.name || '-') + '</td>' +
                    '<td>' + d.address + '</td>' +
                    '<td class="status-' + d.status + '">' + d.status + '</td>' +
                    '<td>' + (d.firmware_version || '-') + '</td>' +
                    '<td>' + (d.location || '-') + '</td>' +
                    '</tr>'
                ).join('');
            } catch (err) {
                console.error('Failed to load devices:', err);
            }
        }
        loadDevices();
    </script>
</body>
</html>
`)
}

func (s *Server) handleUpdatesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Updates - Update Orchestrator</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        h1 { color: #333; }
        .nav { margin: 20px 0; }
        .nav a { margin-right: 15px; color: #007bff; text-decoration: none; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; font-weight: 600; }
        .status { padding: 4px 8px; border-radius: 3px; font-size: 12px; }
        .status-completed { background: #d4edda; color: #155724; }
        .status-in_progress { background: #d1ecf1; color: #0c5460; }
        .status-failed { background: #f8d7da; color: #721c24; }
        .status-pending { background: #fff3cd; color: #856404; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔄 Updates</h1>
        <div class="nav">
            <a href="/">Dashboard</a>
            <a href="/devices">Devices</a>
            <a href="/updates">Updates</a>
        </div>
        <table id="updates-table">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Status</th>
                    <th>Total Devices</th>
                    <th>Completed</th>
                    <th>Failed</th>
                    <th>Progress</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>
    <script>
        async function loadUpdates() {
            try {
                const resp = await fetch('/api/updates');
                const updates = await resp.json();
                const tbody = document.querySelector('#updates-table tbody');
                tbody.innerHTML = updates.map(u => {
                    const progress = u.total_devices > 0
                        ? Math.round((u.completed / u.total_devices) * 100)
                        : 0;
                    return '<tr>' +
                        '<td>' + u.update_id + '</td>' +
                        '<td><span class="status status-' + u.status + '">' + u.status + '</span></td>' +
                        '<td>' + u.total_devices + '</td>' +
                        '<td>' + u.completed + '</td>' +
                        '<td>' + u.failed + '</td>' +
                        '<td>' + progress + '%</td>' +
                        '</tr>';
                }).join('');
            } catch (err) {
                console.error('Failed to load updates:', err);
            }
        }
        loadUpdates();
        setInterval(loadUpdates, 2000); // Refresh every 2 seconds
    </script>
</body>
</html>
`)
}

// API Handlers

func (s *Server) handleDevicesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	devices, err := s.registry.List(ctx, core.Filter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleDeviceAPI(w http.ResponseWriter, r *http.Request) {
	// Extract device ID from path
	deviceID := r.URL.Path[len("/api/devices/"):]
	if deviceID == "" {
		http.Error(w, "device ID required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	device, err := s.registry.Get(ctx, deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func (s *Server) handleUpdatesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all updates from scheduler
	var allUpdates []core.Status

	// Try to get updates in different statuses
	for _, status := range []core.UpdateStatus{
		core.StatusPending,
		core.StatusScheduled,
		core.StatusInProgress,
		core.StatusCompleted,
		core.StatusFailed,
	} {
		updates, _ := s.scheduler.List(ctx, status)
		allUpdates = append(allUpdates, updates...)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allUpdates)
}

func (s *Server) handleUpdateAPI(w http.ResponseWriter, r *http.Request) {
	updateID := r.URL.Path[len("/api/updates/"):]
	if updateID == "" {
		http.Error(w, "update ID required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	status, err := s.scheduler.Status(ctx, updateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update core.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.scheduler.Schedule(ctx, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scheduled"})
}

func (s *Server) handleCancelUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UpdateID string `json:"update_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.scheduler.Cancel(ctx, req.UpdateID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// WebSocket Handler

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	// Unregister on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
	}()

	// Keep connection alive and send periodic updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send update status
			if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
				return
			}
		}
	}
}

// BroadcastUpdate sends an update to all connected WebSocket clients.
func (s *Server) BroadcastUpdate(updateID string, status *core.Status) {
	message, err := json.Marshal(map[string]interface{}{
		"type":      "update",
		"update_id": updateID,
		"status":    status,
	})
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
		}
	}
}
