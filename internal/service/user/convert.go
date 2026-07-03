// Package user provides user management operations.
package user

import (
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

// EntToProto converts an ent User to a proto User.
func EntToProto(u *ent.User) *userpb.User {
	if u == nil {
		return nil
	}
	return &userpb.User{
		Id:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		CreatedAt:   convert.PTime(u.CreatedAt),
		UpdatedAt:   convert.PTime(u.UpdatedAt),
	}
}

// EntDeviceToProto converts an ent Device to a proto Device.
func EntDeviceToProto(d *ent.Device) *userpb.Device {
	if d == nil {
		return nil
	}
	return &userpb.Device{
		DeviceId:      d.DeviceID,
		DeviceName:    d.DeviceName,
		Platform:      d.Platform,
		OsVersion:     d.OsVersion,
		ClientVersion: d.ClientVersion,
		LastSyncAt:    convert.PTimeSafe(d.LastSyncAt),
		RegisteredAt:  convert.PTime(d.CreatedAt),
	}
}
