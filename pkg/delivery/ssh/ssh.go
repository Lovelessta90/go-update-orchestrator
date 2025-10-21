package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Config holds SSH delivery configuration.
type Config struct {
	// Username for SSH authentication
	Username string

	// PrivateKeyPath is the path to the SSH private key file
	PrivateKeyPath string

	// Password for SSH authentication (alternative to key-based auth)
	Password string

	// Port for SSH connection (default: 22)
	Port int

	// Timeout for SSH operations
	Timeout time.Duration

	// RemotePath is the destination path on the device for updates
	RemotePath string

	// VerifyCommand is the SSH command to verify the update (e.g., "/usr/bin/check-version")
	VerifyCommand string

	// KnownHostsPath is the path to known_hosts file (optional)
	KnownHostsPath string
}

// DefaultConfig returns SSH configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Username:   "root",
		Port:       22,
		Timeout:    30 * time.Second,
		RemotePath: "/tmp/update.bin",
	}
}

// Delivery implements SSH-based update delivery.
type Delivery struct {
	config *Config
}

// New creates a new SSH delivery mechanism with default config.
func New() *Delivery {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new SSH delivery mechanism with custom config.
func NewWithConfig(config *Config) *Delivery {
	return &Delivery{config: config}
}

// Push delivers the update payload to a device via SFTP.
func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
	// Create SSH client config
	sshConfig, err := d.createSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Parse device address (format: hostname or hostname:port)
	address := device.Address
	if d.config.Port != 22 {
		address = fmt.Sprintf("%s:%d", device.Address, d.config.Port)
	} else if !hasPort(address) {
		address = fmt.Sprintf("%s:22", address)
	}

	// Connect to SSH server with timeout
	connChan := make(chan *ssh.Client, 1)
	errChan := make(chan error, 1)

	go func() {
		client, err := ssh.Dial("tcp", address, sshConfig)
		if err != nil {
			errChan <- err
			return
		}
		connChan <- client
	}()

	var client *ssh.Client
	select {
	case client = <-connChan:
		defer client.Close()
	case err := <-errChan:
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d.config.Timeout):
		return fmt.Errorf("SSH connection timeout")
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure remote directory exists
	remoteDir := filepath.Dir(d.config.RemotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(d.config.RemotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Stream payload to remote file with context cancellation support
	doneChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(remoteFile, payload)
		doneChan <- err
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			return fmt.Errorf("failed to transfer update: %w", err)
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Verify checks if the update was successfully applied via SSH command.
func (d *Delivery) Verify(ctx context.Context, device core.Device) error {
	if d.config.VerifyCommand == "" {
		// No verification command configured - skip verification
		return nil
	}

	// Create SSH client config
	sshConfig, err := d.createSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Parse device address
	address := device.Address
	if d.config.Port != 22 {
		address = fmt.Sprintf("%s:%d", device.Address, d.config.Port)
	} else if !hasPort(address) {
		address = fmt.Sprintf("%s:22", address)
	}

	// Connect to SSH server
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer client.Close()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Run verification command with context timeout
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- session.Run(d.config.VerifyCommand)
	}()

	select {
	case err := <-cmdDone:
		if err != nil {
			return fmt.Errorf("verification command failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d.config.Timeout):
		return fmt.Errorf("verification timeout")
	}
}

// createSSHConfig creates an SSH client configuration.
func (d *Delivery) createSSHConfig() (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            d.config.Username,
		Timeout:         d.config.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use known_hosts for production
	}

	// Prefer key-based authentication
	if d.config.PrivateKeyPath != "" {
		key, err := os.ReadFile(d.config.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else if d.config.Password != "" {
		// Fall back to password authentication
		config.Auth = []ssh.AuthMethod{
			ssh.Password(d.config.Password),
		}
	} else {
		return nil, fmt.Errorf("no authentication method configured (need PrivateKeyPath or Password)")
	}

	return config, nil
}

// hasPort checks if an address already includes a port.
func hasPort(address string) bool {
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] == ':' {
			return true
		}
		if address[i] == '.' {
			return false
		}
	}
	return false
}
