package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// TestSSHDelivery_Push tests file transfer via SFTP
func TestSSHDelivery_Push(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH test in short mode")
	}

	// Start mock SSH server
	server, privateKeyPath, cleanup := setupMockSSHServer(t)
	defer cleanup()

	// Create SSH delivery with test config
	config := DefaultConfig()
	config.PrivateKeyPath = privateKeyPath
	config.Username = "testuser"
	config.Port = server.Port
	config.RemotePath = "/tmp/test-update.bin"
	config.Timeout = 5 * time.Second

	delivery := NewWithConfig(config)

	// Create test device
	device := core.Device{
		ID:      "test-device",
		Address: "127.0.0.1",
	}

	// Create test payload
	payload := strings.NewReader("test update content")

	// Push update
	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify file was created on server
	content, err := os.ReadFile(filepath.Join(server.TempDir, "tmp", "test-update.bin"))
	if err != nil {
		t.Fatalf("Failed to read uploaded file: %v", err)
	}

	if string(content) != "test update content" {
		t.Errorf("Expected content 'test update content', got '%s'", string(content))
	}
}

func TestSSHDelivery_PushWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH test in short mode")
	}

	server, privateKeyPath, cleanup := setupMockSSHServer(t)
	defer cleanup()

	config := DefaultConfig()
	config.PrivateKeyPath = privateKeyPath
	config.Username = "testuser"
	config.Port = server.Port
	config.Timeout = 5 * time.Second

	delivery := NewWithConfig(config)

	device := core.Device{
		ID:      "test-device",
		Address: "127.0.0.1",
	}

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	payload := strings.NewReader("test content")

	// Push should fail with context canceled
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Error("Expected error with canceled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}

func TestSSHDelivery_PasswordAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH test in short mode")
	}

	server, _, cleanup := setupMockSSHServer(t)
	defer cleanup()

	// Configure with password auth
	config := DefaultConfig()
	config.Password = "testpass"
	config.Username = "testuser"
	config.Port = server.Port
	config.RemotePath = "/tmp/password-test.bin"

	delivery := NewWithConfig(config)

	device := core.Device{
		ID:      "test-device",
		Address: "127.0.0.1",
	}

	payload := strings.NewReader("password auth test")

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err != nil {
		t.Fatalf("Push with password auth failed: %v", err)
	}
}

func TestSSHDelivery_ConfigValidation(t *testing.T) {
	config := DefaultConfig()
	// No auth method configured
	config.PrivateKeyPath = ""
	config.Password = ""

	delivery := NewWithConfig(config)

	device := core.Device{
		ID:      "test-device",
		Address: "127.0.0.1",
	}

	payload := strings.NewReader("test")

	ctx := context.Background()
	err := delivery.Push(ctx, device, payload)
	if err == nil {
		t.Error("Expected error with no auth method, got nil")
	}
	if !strings.Contains(err.Error(), "no authentication method") {
		t.Errorf("Expected no auth method error, got: %v", err)
	}
}

func TestHasPort(t *testing.T) {
	tests := []struct {
		address  string
		expected bool
	}{
		{"192.168.1.1", false},
		{"192.168.1.1:22", true},
		{"example.com", false},
		{"example.com:2222", true},
		{"localhost", false},
		{"localhost:22", true},
	}

	for _, tt := range tests {
		result := hasPort(tt.address)
		if result != tt.expected {
			t.Errorf("hasPort(%s) = %v, expected %v", tt.address, result, tt.expected)
		}
	}
}

// Mock SSH Server for testing

type MockSSHServer struct {
	listener  net.Listener
	Port      int
	TempDir   string
	sshConfig *ssh.ServerConfig
}

func setupMockSSHServer(t *testing.T) (*MockSSHServer, string, func()) {
	t.Helper()

	// Create temp directory for file storage
	tempDir := t.TempDir()

	// Generate test SSH key
	privateKeyPath := filepath.Join(tempDir, "test_key")

	// Create a simple test key (insecure, for testing only)
	privateKey := []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAYEAw8s1lD8YkLDpQKz4VPZ0fH3F0rKqG5FJ1aMPXZL8qJPTy+hL0pKJ
qQh2EcLdZ6vZpO2fGdQqNKG3WLKqJPKLZ6hCQJ8KqPLdZ6vZpO2fGdQqNKG3WLKqJPKLZ6
hCQJ8KqPLdZ6vZpO2fGdQqNKG3WLKqJPKLZ6hCQJ8KqPLdZ6vZpO2fGdQqNKG3WLKqJPKL
Z6hCQJ8KqPLdZ6vZpO2fGdQqNKG3WLKqJPKLZ6hCQJ8KqPL=
-----END OPENSSH PRIVATE KEY-----`)

	if err := os.WriteFile(privateKeyPath, privateKey, 0600); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}

	// Parse private key for server
	hostKey, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to parse host key: %v", err)
	}

	// Configure SSH server
	config := &ssh.ServerConfig{
		NoClientAuth: false,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == "testuser" && string(password) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Accept any public key for testing
			return nil, nil
		},
	}
	config.AddHostKey(hostKey)

	// Start SSH server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	server := &MockSSHServer{
		listener:  listener,
		Port:      port,
		TempDir:   tempDir,
		sshConfig: config,
	}

	// Start accepting connections
	go server.acceptConnections(t)

	cleanup := func() {
		listener.Close()
	}

	return server, privateKeyPath, cleanup
}

func (s *MockSSHServer) acceptConnections(t *testing.T) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // Server closed
		}

		go s.handleConnection(t, conn)
	}
}

func (s *MockSSHServer) handleConnection(t *testing.T, netConn net.Conn) {
	defer netConn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshConfig)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels (subsystems like SFTP)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}

		go func() {
			for req := range requests {
				if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
					req.Reply(true, nil)

					// Start SFTP server
					server, err := sftp.NewServer(channel, sftp.WithServerWorkingDirectory(s.TempDir))
					if err != nil {
						return
					}
					server.Serve()
					return
				}
				req.Reply(false, nil)
			}
		}()
	}
}
