package core

import "time"

// DeviceStatus represents the connectivity state of a device.
type DeviceStatus string

const (
	DeviceOnline  DeviceStatus = "online"  // Currently connected and reachable
	DeviceOffline DeviceStatus = "offline" // Not currently reachable
	DeviceUnknown DeviceStatus = "unknown" // Never seen or status unclear
)

// Device represents a target device for updates.
type Device struct {
	ID              string            // Unique device identifier
	Name            string            // Human-readable name
	Address         string            // Network address (IP, hostname, URL)
	Status          DeviceStatus      // Current connectivity status
	LastSeen        *time.Time        // Last time device was online
	FirmwareVersion string            // Current firmware version
	Location        string            // Physical location (store, region, etc)
	Metadata        map[string]string // Custom device metadata (tags, groups)
	CreatedAt       time.Time         // When device was registered
	UpdatedAt       time.Time         // Last metadata update
}

// Filter represents criteria for selecting devices.
type Filter struct {
	IDs             []string          // Filter by specific device IDs
	Status          *DeviceStatus     // Filter by connectivity status (online/offline)
	Location        string            // Filter by location
	MinFirmware     string            // Filter devices with firmware >= this version
	MaxFirmware     string            // Filter devices with firmware <= this version
	Tags            map[string]string // Filter by metadata tags
	LastSeenBefore  *time.Time        // Filter devices last seen before this time
	LastSeenAfter   *time.Time        // Filter devices last seen after this time
	Limit           int               // Maximum number of devices to return
	Offset          int               // Pagination offset
}
