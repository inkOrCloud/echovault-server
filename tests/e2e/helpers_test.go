// Package e2e_test contains shared test utilities for EchoVault end-to-end tests.
package e2e_test

import (
	"context"
	"google.golang.org/grpc/metadata"
)

const testJWTSecret = "e2e-test-secret-key"

// authCtx creates a context with the Bearer token in gRPC metadata.
func authCtx(token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(context.Background(), md)
}
