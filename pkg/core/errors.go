package core

import "errors"

var (
	// ErrDeviceNotFound indicates a device was not found in the registry.
	ErrDeviceNotFound = errors.New("device not found")

	// ErrUpdateNotFound indicates an update was not found.
	ErrUpdateNotFound = errors.New("update not found")

	// ErrUpdateInProgress indicates an update is already in progress for a device.
	ErrUpdateInProgress = errors.New("update already in progress")

	// ErrDeliveryFailed indicates the delivery mechanism failed.
	ErrDeliveryFailed = errors.New("delivery failed")

	// ErrInvalidDevice indicates device validation failed.
	ErrInvalidDevice = errors.New("invalid device")

	// ErrInvalidUpdate indicates update validation failed.
	ErrInvalidUpdate = errors.New("invalid update")

	// ErrCancelled indicates the operation was cancelled.
	ErrCancelled = errors.New("operation cancelled")
)
