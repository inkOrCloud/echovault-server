package user

import (
	"context"
	"errors"
	"time"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/device"
)

func (s *Service) RegisterDevice(ctx context.Context, userID, deviceID, name, platform string) error {
	exists, _ := s.client.Device.Query().Where(device.DeviceID(deviceID)).Exist(ctx)
	if exists {
		return errors.New("device already registered")
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
	return err
}

func (s *Service) ListDevices(ctx context.Context, userID string) ([]*ent.Device, error) {
	return s.client.Device.Query().Where(device.UserID(userID)).All(ctx)
}

func (s *Service) RemoveDevice(ctx context.Context, userID, deviceID string) error {
	n, err := s.client.Device.Delete().Where(
		device.DeviceID(deviceID),
		device.UserID(userID),
	).Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("device not found")
	}
	return nil
}
