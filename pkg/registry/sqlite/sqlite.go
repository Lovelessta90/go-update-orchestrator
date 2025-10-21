package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

// Registry implements a SQLite-based device registry.
type Registry struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS devices (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	address TEXT NOT NULL,
	status TEXT NOT NULL,
	last_seen DATETIME,
	firmware_version TEXT,
	location TEXT,
	metadata TEXT, -- JSON encoded map[string]string
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

-- Index for common query patterns
CREATE INDEX IF NOT EXISTS idx_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_location ON devices(location);
CREATE INDEX IF NOT EXISTS idx_firmware ON devices(firmware_version);
CREATE INDEX IF NOT EXISTS idx_last_seen ON devices(last_seen);
CREATE INDEX IF NOT EXISTS idx_updated_at ON devices(updated_at);
`

// New creates a new SQLite registry.
func New(dbPath string) (*Registry, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Registry{db: db}, nil
}

// Close closes the database connection.
func (r *Registry) Close() error {
	return r.db.Close()
}

// List returns devices matching the given filter.
func (r *Registry) List(ctx context.Context, filter core.Filter) ([]core.Device, error) {
	query, args := buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]core.Device, 0)
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}

		// Apply metadata tag filtering (post-query since metadata is JSON)
		if !matchesMetadataTags(device.Metadata, filter.Tags) {
			continue
		}

		devices = append(devices, device)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating devices: %w", err)
	}

	return devices, nil
}

// buildListQuery constructs a SQL query based on the filter.
func buildListQuery(filter core.Filter) (string, []interface{}) {
	query := "SELECT id, name, address, status, last_seen, firmware_version, location, metadata, created_at, updated_at FROM devices WHERE 1=1"
	args := make([]interface{}, 0)

	// Filter by specific IDs
	if len(filter.IDs) > 0 {
		placeholders := make([]string, len(filter.IDs))
		for i, id := range filter.IDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += " AND id IN (" + strings.Join(placeholders, ",") + ")"
	}

	// Filter by status
	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filter.Status))
	}

	// Filter by location
	if filter.Location != "" {
		query += " AND location = ?"
		args = append(args, filter.Location)
	}

	// Filter by firmware version (simple string comparison for now)
	if filter.MinFirmware != "" {
		query += " AND firmware_version >= ?"
		args = append(args, filter.MinFirmware)
	}

	if filter.MaxFirmware != "" {
		query += " AND firmware_version <= ?"
		args = append(args, filter.MaxFirmware)
	}

	// Filter by last seen time
	if filter.LastSeenBefore != nil {
		query += " AND last_seen < ?"
		args = append(args, filter.LastSeenBefore.Format(time.RFC3339))
	}

	if filter.LastSeenAfter != nil {
		query += " AND last_seen > ?"
		args = append(args, filter.LastSeenAfter.Format(time.RFC3339))
	}

	// Pagination
	query += " ORDER BY id"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	return query, args
}

// matchesMetadataTags checks if device metadata matches all required tags.
func matchesMetadataTags(metadata, tags map[string]string) bool {
	if len(tags) == 0 {
		return true
	}

	for key, value := range tags {
		if deviceValue, ok := metadata[key]; !ok || deviceValue != value {
			return false
		}
	}
	return true
}

// scanDevice scans a row into a Device struct.
func scanDevice(row interface {
	Scan(dest ...interface{}) error
}) (core.Device, error) {
	var device core.Device
	var lastSeenStr sql.NullString
	var metadataJSON sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&device.ID,
		&device.Name,
		&device.Address,
		&device.Status,
		&lastSeenStr,
		&device.FirmwareVersion,
		&device.Location,
		&metadataJSON,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		// Return sql.ErrNoRows unwrapped so it can be detected
		if errors.Is(err, sql.ErrNoRows) {
			return core.Device{}, err
		}
		return core.Device{}, fmt.Errorf("failed to scan device: %w", err)
	}

	// Parse last seen
	if lastSeenStr.Valid {
		t, err := time.Parse(time.RFC3339, lastSeenStr.String)
		if err != nil {
			return core.Device{}, fmt.Errorf("failed to parse last_seen: %w", err)
		}
		device.LastSeen = &t
	}

	// Parse metadata JSON
	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &device.Metadata); err != nil {
			return core.Device{}, fmt.Errorf("failed to parse metadata: %w", err)
		}
	} else {
		device.Metadata = make(map[string]string)
	}

	// Parse timestamps
	device.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return core.Device{}, fmt.Errorf("failed to parse created_at: %w", err)
	}

	device.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return core.Device{}, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	return device, nil
}

// Get retrieves a single device by ID.
func (r *Registry) Get(ctx context.Context, id string) (*core.Device, error) {
	query := "SELECT id, name, address, status, last_seen, firmware_version, location, metadata, created_at, updated_at FROM devices WHERE id = ?"

	row := r.db.QueryRowContext(ctx, query, id)
	device, err := scanDevice(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrDeviceNotFound
		}
		return nil, err
	}

	return &device, nil
}

// Add registers a new device.
func (r *Registry) Add(ctx context.Context, device core.Device) error {
	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Set timestamps if not already set
	now := time.Now()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	if device.UpdatedAt.IsZero() {
		device.UpdatedAt = now
	}

	query := `
		INSERT INTO devices (id, name, address, status, last_seen, firmware_version, location, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastSeenStr sql.NullString
	if device.LastSeen != nil {
		lastSeenStr.String = device.LastSeen.Format(time.RFC3339)
		lastSeenStr.Valid = true
	}

	_, err = r.db.ExecContext(ctx, query,
		device.ID,
		device.Name,
		device.Address,
		device.Status,
		lastSeenStr,
		device.FirmwareVersion,
		device.Location,
		string(metadataJSON),
		device.CreatedAt.Format(time.RFC3339),
		device.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert device: %w", err)
	}

	return nil
}

// Update modifies an existing device.
func (r *Registry) Update(ctx context.Context, device core.Device) error {
	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update the updated_at timestamp
	device.UpdatedAt = time.Now()

	query := `
		UPDATE devices
		SET name = ?, address = ?, status = ?, last_seen = ?, firmware_version = ?, location = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	var lastSeenStr sql.NullString
	if device.LastSeen != nil {
		lastSeenStr.String = device.LastSeen.Format(time.RFC3339)
		lastSeenStr.Valid = true
	}

	result, err := r.db.ExecContext(ctx, query,
		device.Name,
		device.Address,
		device.Status,
		lastSeenStr,
		device.FirmwareVersion,
		device.Location,
		string(metadataJSON),
		device.UpdatedAt.Format(time.RFC3339),
		device.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return core.ErrDeviceNotFound
	}

	return nil
}

// Delete removes a device from the registry.
func (r *Registry) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM devices WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return core.ErrDeviceNotFound
	}

	return nil
}
