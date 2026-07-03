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
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
)

func newLyricTestServer(t *testing.T) (lyricpb.LyricServiceClient, func()) { //nolint:ireturn
	t.Helper()
	name := "file:lyric_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	svc := lyric.NewService(client)
	handler := evgrpc.NewLyricHandler(svc)
	s := grpc.NewServer()
	lyricpb.RegisterLyricServiceServer(s, handler)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	c := lyricpb.NewLyricServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestLyricSaveAndGetHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newLyricTestServer(t)
	defer cleanup()

	_, err := c.SaveLyric(context.Background(), &lyricpb.SaveLyricRequest{
		SongId: "song-1", Content: "[00:01.00]test",
		Type: lyricpb.Lyric_TYPE_ORIGINAL, Language: "zh",
	})
	require.NoError(t, err)

	resp, err := c.GetLyric(context.Background(), &lyricpb.GetLyricRequest{SongId: "song-1"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetLyrics())
	require.Equal(t, "[00:01.00]test", resp.GetLyrics()[0].GetContent())
}
