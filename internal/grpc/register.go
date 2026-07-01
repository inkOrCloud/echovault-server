package grpc

import (
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	"google.golang.org/grpc"
)

func RegisterAll(s *grpc.Server, client *ent.Client, jwtSecret string) {
	userSvc := user.NewService(client, jwtSecret)
	userHandler := NewUserHandler(userSvc)
	userpb.RegisterUserServiceServer(s, userHandler)

	syncSvc := sync.NewService(client)
	syncHandler := NewSyncHandler(syncSvc)
	syncpb.RegisterSyncServiceServer(s, syncHandler)

	songSvc := song.NewService(client)
	songHandler := NewSongHandler(songSvc)
	songpb.RegisterSongServiceServer(s, songHandler)
}

func AuthInterceptorOpts(secret string) grpc.ServerOption {
	return grpc.UnaryInterceptor(AuthInterceptor(secret))
}
