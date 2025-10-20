package validation

import (
	"errors"
	"net/url"
	"strings"

	"github.com/dovaclean/go-update-orchestrator/pkg/core"
)

var (
	ErrEmptyDeviceID   = errors.New("device ID cannot be empty")
	ErrEmptyUpdateID   = errors.New("update ID cannot be empty")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrEmptyAddress    = errors.New("device address cannot be empty")
)

// ValidateDevice checks if a device is valid.
func ValidateDevice(device core.Device) error {
	if strings.TrimSpace(device.ID) == "" {
		return ErrEmptyDeviceID
	}
	if strings.TrimSpace(device.Address) == "" {
		return ErrEmptyAddress
	}
	return nil
}

// ValidateUpdate checks if an update is valid.
func ValidateUpdate(update core.Update) error {
	if strings.TrimSpace(update.ID) == "" {
		return ErrEmptyUpdateID
	}
	if strings.TrimSpace(update.PayloadURL) == "" {
		return ErrInvalidURL
	}
	if _, err := url.Parse(update.PayloadURL); err != nil {
		return ErrInvalidURL
	}
	if len(update.DeviceIDs) == 0 {
		return errors.New("update must target at least one device")
	}
	return nil
}

// ValidateDeviceID checks if a device ID is valid.
func ValidateDeviceID(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrEmptyDeviceID
	}
	return nil
}

// ValidateUpdateID checks if an update ID is valid.
func ValidateUpdateID(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrEmptyUpdateID
	}
	return nil
}
