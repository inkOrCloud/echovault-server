// Package grpc provides gRPC server handlers and interceptors for EchoVault.
package grpc

import (
	"context"
	"slices"
	"strings"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey string

const (
	ctxKeyUserID   ctxKey = "user_id"
	ctxKeyDeviceID ctxKey = "device_id"
)

// GetUserID returns the user ID from the context.
func GetUserID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	if !ok {
		return ""
	}
	return v
}

// GetDeviceID returns the device ID from the context.
func GetDeviceID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyDeviceID).(string)
	if !ok {
		return ""
	}
	return v
}

// AuthInterceptor returns a unary server interceptor that validates JWT tokens.
func AuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if isPublicRPC(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		tokenStr := strings.TrimPrefix(authHeader[0], "Bearer ")
		claims, err := auth.ValidateToken(secret, tokenStr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token: "+err.Error())
		}

		ctx = context.WithValue(ctx, ctxKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxKeyDeviceID, claims.DeviceID)
		return handler(ctx, req)
	}
}

func isPublicRPC(method string) bool {
	public := []string{
		"/echo_vault.user.v1.UserService/Register",
		"/echo_vault.user.v1.UserService/Login",
		"/echo_vault.user.v1.UserService/GetServerInfo",
	}
	return slices.Contains(public, method)
}
