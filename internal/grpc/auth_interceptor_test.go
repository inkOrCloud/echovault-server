package grpc_test

import (
	"context"
	"time"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/auth"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetUserID_Empty(t *testing.T) {
	t.Parallel()
	if id := grpc.GetUserID(context.Background()); id != "" {
		t.Errorf("GetUserID = %q, want empty", id)
	}
}

func TestGetDeviceID_Empty(t *testing.T) {
	t.Parallel()
	if id := grpc.GetDeviceID(context.Background()); id != "" {
		t.Errorf("GetDeviceID = %q, want empty", id)
	}
}

func TestIsPublicRPC(t *testing.T) {
	t.Parallel()
	public := []string{
		"/echo_vault.user.v1.UserService/Register",
		"/echo_vault.user.v1.UserService/Login",
		"/echo_vault.user.v1.UserService/GetServerInfo",
	}
	for _, method := range public {
		interceptor := grpc.AuthInterceptor("test-secret")
		handlerCalled := false
		handler := func(ctx context.Context, req any) (any, error) {
			handlerCalled = true
			return "ok", nil
		}
		_, err := interceptor(context.Background(), nil, &ggrpc.UnaryServerInfo{FullMethod: method}, handler)
		if err != nil {
			t.Errorf("public RPC %q failed: %v", method, err)
		}
		if !handlerCalled {
			t.Errorf("handler not called for public RPC %q", method)
		}
	}
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	t.Parallel()
	interceptor := grpc.AuthInterceptor("test-secret")
	_, err := interceptor(context.Background(), nil,
		&ggrpc.UnaryServerInfo{FullMethod: "/echo_vault.song.v1.SongService/GetSong"},
		func(ctx context.Context, req any) (any, error) { return "ok", nil })
	if err == nil {
		t.Fatal("expected error for missing metadata")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("status code = %v, want Unauthenticated", status.Code(err))
	}
}

func TestAuthInterceptor_MissingAuthHeader(t *testing.T) {
	t.Parallel()
	interceptor := grpc.AuthInterceptor("test-secret")
	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := interceptor(ctx, nil,
		&ggrpc.UnaryServerInfo{FullMethod: "/echo_vault.song.v1.SongService/GetSong"},
		func(ctx context.Context, req any) (any, error) { return "ok", nil })
	if err == nil {
		t.Fatal("expected error for missing auth header")
	}
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	t.Parallel()
	interceptor := grpc.AuthInterceptor("test-secret")
	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := interceptor(ctx, nil,
		&ggrpc.UnaryServerInfo{FullMethod: "/echo_vault.song.v1.SongService/GetSong"},
		func(ctx context.Context, req any) (any, error) { return "ok", nil })
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	t.Parallel()
	secret := "test-secret"
	token, err := auth.GenerateToken(secret, "user-123", "dev-456", time.Hour * 24 * 365)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	interceptor := grpc.AuthInterceptor(secret)
	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedUserID, capturedDeviceID string
	handler := func(ctx context.Context, req any) (any, error) {
		capturedUserID = grpc.GetUserID(ctx)
		capturedDeviceID = grpc.GetDeviceID(ctx)
		return "ok", nil
	}

	_, err = interceptor(ctx, nil,
		&ggrpc.UnaryServerInfo{FullMethod: "/echo_vault.song.v1.SongService/GetSong"},
		handler)
	if err != nil {
		t.Fatalf("AuthInterceptor error: %v", err)
	}
	if capturedUserID != "user-123" {
		t.Errorf("UserID = %q, want user-123", capturedUserID)
	}
	if capturedDeviceID != "dev-456" {
		t.Errorf("DeviceID = %q, want dev-456", capturedDeviceID)
	}
}
