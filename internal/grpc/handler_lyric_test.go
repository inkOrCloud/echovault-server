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
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
)

func newLyricTestServer(t *testing.T) (lyricpb.LyricServiceClient, func()) {
	t.Helper()
	drv, _ := entsql.Open("sqlite3", "file:lyric_handler?mode=memory&cache=shared&_fk=1")
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	client.Schema.Create(context.Background())
	svc := lyric.NewService(client)
	handler := evgrpc.NewLyricHandler(svc)
	s := grpc.NewServer()
	lyricpb.RegisterLyricServiceServer(s, handler)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go s.Serve(lis)
	conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	c := lyricpb.NewLyricServiceClient(conn)
	return c, func() { conn.Close(); s.GracefulStop() }
}

func TestLyricSaveAndGetHandler(t *testing.T) {
	c, cleanup := newLyricTestServer(t); defer cleanup()

	_, err := c.SaveLyric(context.Background(), &lyricpb.SaveLyricRequest{
		SongId: "song-1", Content: "[00:01.00]test",
		Type: lyricpb.Lyric_TYPE_ORIGINAL, Language: "zh",
	})
	if err != nil { t.Fatalf("SaveLyric RPC error = %v", err) }

	resp, err := c.GetLyric(context.Background(), &lyricpb.GetLyricRequest{SongId: "song-1"})
	if err != nil { t.Fatalf("GetLyric RPC error = %v", err) }
	if len(resp.Lyrics) == 0 { t.Fatal("no lyrics returned") }
	if resp.Lyrics[0].Content != "[00:01.00]test" { t.Errorf("content mismatch") }
}
