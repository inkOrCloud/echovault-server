package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
)

func newPlaylistTestServer(t *testing.T) (playlistpb.PlaylistServiceClient, func()) { //nolint:ireturn
	t.Helper()
	name := "file:pl_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	svc := playlist.NewService(client)
	handler := evgrpc.NewPlaylistHandler(svc)
	s := grpc.NewServer()
	playlistpb.RegisterPlaylistServiceServer(s, handler)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	c := playlistpb.NewPlaylistServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestPlaylistCreateAndGetHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newPlaylistTestServer(t)
	defer cleanup()

	resp, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "My List",
	})
	require.NoError(t, err)
	require.Equal(t, "My List", resp.GetPlaylist().GetName())
}
