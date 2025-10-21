package web

import (
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
	"github.com/dovaclean/go-update-orchestrator/pkg/orchestrator"
	"github.com/dovaclean/go-update-orchestrator/pkg/registry"
	"github.com/dovaclean/go-update-orchestrator/pkg/scheduler"
)

//go:embed templates/* static/*
var embeddedFS embed.FS

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
	tmpl, err := template.ParseFS(embeddedFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Server{
		addr:         config.Address,
		orchestrator: orch,
		scheduler:    sched,
		registry:     reg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients:   make(map[*websocket.Conn]bool),
		templates: tmpl,
	}, nil
}

// Start starts the web server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve static files
	staticFS := http.FS(embeddedFS)
	mux.Handle("/static/", http.StripPrefix("/", http.FileServer(staticFS)))

	// Pages
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/devices", s.handleDevices)
	mux.HandleFunc("/updates", s.handleUpdates)

	// API endpoints
	mux.HandleFunc("/api/devices", s.handleDevicesAPI)
	mux.HandleFunc("/api/updates", s.handleUpdatesAPI)
	mux.HandleFunc("/api/updates/schedule", s.handleScheduleUpdate)
	mux.HandleFunc("/api/updates/cancel", s.handleCancelUpdate)

	// WebSocket
	mux.HandleFunc("/ws", s.handleWebSocket)

	log.Printf("Web UI starting on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// Page Handlers

type pageData struct {
	Title string
	Page  string
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := pageData{
		Title: "Dashboard",
		Page:  "dashboard",
	}

	// Parse base + dashboard templates together
	tmpl, err := template.ParseFS(embeddedFS, "templates/base.html", "templates/dashboard.html")
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := pageData{
		Title: "Devices",
		Page:  "devices",
	}

	// Parse base + devices templates together
	tmpl, err := template.ParseFS(embeddedFS, "templates/base.html", "templates/devices.html")
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) handleUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := pageData{
		Title: "Updates",
		Page:  "updates",
	}

	// Parse base + updates templates together
	tmpl, err := template.ParseFS(embeddedFS, "templates/base.html", "templates/updates.html")
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// API Handlers

func (s *Server) handleDevicesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	devices, err := s.registry.List(ctx, core.Filter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleUpdatesAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// List all updates
	updates, err := s.scheduler.ListAll(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(updates)
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// WebSocket Handler

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
	}()

	// Keep connection alive
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Broadcast sends a message to all connected WebSocket clients.
func (s *Server) Broadcast(message interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket write error: %v", err)
		}
	}
}
