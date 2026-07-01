package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
)

func newSongTestServer(t *testing.T) (songpb.SongServiceClient, func()) {
	t.Helper()
	drv, _ := entsql.Open("sqlite3", "file:song_handler?mode=memory&cache=shared&_fk=1")
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	svc := song.NewService(client)
	handler := evgrpc.NewSongHandler(svc)

	s := grpc.NewServer()
	songpb.RegisterSongServiceServer(s, handler)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go s.Serve(lis)
	conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	songClient := songpb.NewSongServiceClient(conn)

	return songClient, func() { conn.Close(); s.GracefulStop() }
}

func TestSongCheckByHashHandler(t *testing.T) {
	client, cleanup := newSongTestServer(t)
	defer cleanup()

	// 先发布
	client.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title: "Hash Test", FileHash: "hash123",
	})

	resp, err := client.CheckSongsByHash(context.Background(), &songpb.CheckSongsByHashRequest{
		FileHashes: []string{"hash123", "nonexistent"},
	})
	if err != nil {
		t.Fatalf("CheckSongsByHash RPC error = %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(resp.Results))
	}
	if !resp.Results[0].Exists {
		t.Error("results[0].Exists = false, want true")
	}
	if resp.Results[1].Exists {
		t.Error("results[1].Exists = true, want false")
	}
}

func TestSongPublishHandler(t *testing.T) {
	client, cleanup := newSongTestServer(t)
	defer cleanup()

	resp, err := client.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title: "New Song", Artist: "Me", FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong RPC error = %v", err)
	}
	if resp.Song.Id == "" {
		t.Error("Song.Id is empty")
	}
	if resp.Song.Title != "New Song" {
		t.Errorf("Title = %q, want %q", resp.Song.Title, "New Song")
	}
}
