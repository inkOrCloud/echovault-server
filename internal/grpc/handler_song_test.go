package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newSongTestServer(t *testing.T) (songpb.SongServiceClient, func()) {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:song_handler?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	err = client.Schema.Create(context.Background())
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	svc := song.NewService(client)
	handler := evgrpc.NewSongHandler(svc)
	s := grpc.NewServer()
	songpb.RegisterSongServiceServer(s, handler)
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	songClient := songpb.NewSongServiceClient(conn)
	return songClient, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestSongCheckByHashHandler(t *testing.T) {
	t.Parallel()
	client, cleanup := newSongTestServer(t)
	defer cleanup()

	_, _ = client.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title: "Hash Test", FileHash: "hash123",
	})

	resp, err := client.CheckSongsByHash(context.Background(), &songpb.CheckSongsByHashRequest{
		FileHashes: []string{"hash123", "nonexistent"},
	})
	if err != nil {
		t.Fatalf("CheckSongsByHash RPC error = %v", err)
	}
	if len(resp.GetResults()) != 2 {
		t.Fatalf("results = %d, want 2", len(resp.GetResults()))
	}
	if !resp.GetResults()[0].GetExists() {
		t.Error("results[0].Exists = false, want true")
	}
	if resp.GetResults()[1].GetExists() {
		t.Error("results[1].Exists = true, want false")
	}
}

func TestSongPublishHandler(t *testing.T) {
	t.Parallel()
	client, cleanup := newSongTestServer(t)
	defer cleanup()

	resp, err := client.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title: "New Song", Artist: "Me", FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong RPC error = %v", err)
	}
	if resp.GetSong().GetId() == "" {
		t.Error("PublishSong returned empty ID")
	}
}
