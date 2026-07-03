package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/device"
)

// ErrDeviceAlreadyRegistered indicates the device ID is already registered.
var ErrDeviceAlreadyRegistered = errors.New("device already registered")

// ErrDeviceNotFound indicates the device was not found.
var ErrDeviceNotFound = errors.New("device not found")

// RegisterDevice registers a new device for a user.
func (s *Service) RegisterDevice(ctx context.Context, userID, deviceID, name, platform string) error {
	exists, _ := s.client.Device.Query().Where(device.DeviceID(deviceID)).Exist(ctx)
	if exists {
		return ErrDeviceAlreadyRegistered
	}

	_, err := s.client.Device.Create().
		SetDeviceID(deviceID).
		SetDeviceName(name).
		SetPlatform(platform).
		SetUserID(userID).
		SetLastSyncAt(time.Now()).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}
	return nil
}

// ListDevices returns all devices registered to a user.
func (s *Service) ListDevices(ctx context.Context, userID string) ([]*ent.Device, error) {
	devices, err := s.client.Device.Query().Where(device.UserID(userID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

// RemoveDevice removes a device registration.
func (s *Service) RemoveDevice(ctx context.Context, userID, deviceID string) error {
	n, err := s.client.Device.Delete().Where(
		device.DeviceID(deviceID),
		device.UserID(userID),
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	if n == 0 {
		return ErrDeviceNotFound
	}
	return nil
}
