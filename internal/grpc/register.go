package grpc

import (
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	"google.golang.org/grpc"
)

// RegisterAll registers all gRPC service handlers on the server.
func RegisterAll(s *grpc.Server, client *ent.Client, jwtSecret string) {
	userSvc := user.NewService(client, jwtSecret)
	userpb.RegisterUserServiceServer(s, NewUserHandler(userSvc))

	syncSvc := sync.NewService(client)
	syncpb.RegisterSyncServiceServer(s, NewSyncHandler(syncSvc))

	songSvc := song.NewService(client)
	songpb.RegisterSongServiceServer(s, NewSongHandler(songSvc))

	lyricSvc := lyric.NewService(client)
	lyricpb.RegisterLyricServiceServer(s, NewLyricHandler(lyricSvc))

	playlistSvc := playlist.NewService(client)
	playlistpb.RegisterPlaylistServiceServer(s, NewPlaylistHandler(playlistSvc))
}
