package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
)

func newPlaylistTestServer(t *testing.T) (playlistpb.PlaylistServiceClient, func()) {
	t.Helper()
	drv, _ := entsql.Open("sqlite3", "file:pl_handler?mode=memory&cache=shared&_fk=1")
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	client.Schema.Create(context.Background())
	svc := playlist.NewService(client)
	handler := evgrpc.NewPlaylistHandler(svc)
	s := grpc.NewServer()
	playlistpb.RegisterPlaylistServiceServer(s, handler)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go s.Serve(lis)
	conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	c := playlistpb.NewPlaylistServiceClient(conn)
	return c, func() { conn.Close(); s.GracefulStop() }
}

func TestPlaylistCreateAndGetHandler(t *testing.T) {
	c, cleanup := newPlaylistTestServer(t); defer cleanup()

	resp, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "My List",
	})
	if err != nil { t.Fatalf("CreatePlaylist RPC error = %v", err) }
	if resp.Playlist.Name != "My List" { t.Errorf("Name = %q", resp.Playlist.Name) }
}
