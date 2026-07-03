package grpc

import (
	"context"

	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserHandler implements the UserService gRPC server.
type UserHandler struct {
	userpb.UnimplementedUserServiceServer

	svc *user.Service
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *user.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register creates a new user account.
func (h *UserHandler) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	resp, err := h.svc.Register(ctx, req.GetUsername(), req.GetPassword(), req.GetDisplayName())
	if err != nil {
		code := codes.InvalidArgument
		if err.Error() == "username already exists" {
			code = codes.AlreadyExists
		}
		return nil, status.Error(code, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	u, err := h.svc.GetUser(ctx, resp.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get created user") //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &userpb.RegisterResponse{
		User:        convertUser(u),
		AccessToken: resp.Token,
	}, nil
}

// Login authenticates a user and returns a token.
func (h *UserHandler) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	resp, err := h.svc.Login(ctx, req.GetUsername(), req.GetPassword(), req.GetDeviceId())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	u, err := h.svc.GetUser(ctx, resp.UserID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user after login") //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &userpb.LoginResponse{
		User:        convertUser(u),
		AccessToken: resp.Token,
	}, nil
}

// GetCurrentUser returns the currently authenticated user.
func (h *UserHandler) GetCurrentUser(ctx context.Context, _ *userpb.GetCurrentUserRequest) (*userpb.GetCurrentUserResponse, error) {
	userID := GetUserID(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "not authenticated") //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	u, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &userpb.GetCurrentUserResponse{User: convertUser(u)}, nil
}

// ListDevices returns all devices registered for the authenticated user.
func (h *UserHandler) ListDevices(ctx context.Context, _ *userpb.ListDevicesRequest) (*userpb.ListDevicesResponse, error) {
	userID := GetUserID(ctx)
	devices, err := h.svc.ListDevices(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	pbDevices := make([]*userpb.Device, len(devices))
	for i, d := range devices {
		pbDevices[i] = convertDevice(d)
	}
	return &userpb.ListDevicesResponse{Devices: pbDevices}, nil
}

// RegisterDevice registers a new device for the authenticated user.
func (h *UserHandler) RegisterDevice(ctx context.Context, req *userpb.RegisterDeviceRequest) (*userpb.RegisterDeviceResponse, error) {
	userID := GetUserID(ctx)
	err := h.svc.RegisterDevice(ctx, userID, req.GetDeviceId(), req.GetDeviceName(), req.GetPlatform())
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &userpb.RegisterDeviceResponse{}, nil
}

// RemoveDevice removes a device registered for the authenticated user.
func (h *UserHandler) RemoveDevice(ctx context.Context, req *userpb.RemoveDeviceRequest) (*userpb.RemoveDeviceResponse, error) {
	userID := GetUserID(ctx)
	err := h.svc.RemoveDevice(ctx, userID, req.GetDeviceId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error()) //nolint:wrapcheck // gRPC status errors are intentionally unwrapped
	}
	return &userpb.RemoveDeviceResponse{}, nil
}

func convertUser(u *ent.User) *userpb.User {
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

func convertDevice(d *ent.Device) *userpb.Device {
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
