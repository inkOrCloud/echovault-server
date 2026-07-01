package grpc

import (
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"google.golang.org/grpc"
)

func RegisterAll(s *grpc.Server, client *ent.Client, jwtSecret string) {
	userSvc := user.NewService(client, jwtSecret)
	userHandler := NewUserHandler(userSvc)
	userpb.RegisterUserServiceServer(s, userHandler)
}

func AuthInterceptorOpts(secret string) grpc.ServerOption {
	return grpc.UnaryInterceptor(AuthInterceptor(secret))
}
