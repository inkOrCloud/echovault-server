package song_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:song?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func TestCheckSongsByHash_Empty(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	results, err := svc.CheckSongsByHash(ctx, []string{"hash-nonexistent"})
	if err != nil {
		t.Fatalf("CheckSongsByHash() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Exists {
		t.Error("result.Exists = true for unknown hash, want false")
	}
}

func TestCheckSongsByHash_Found(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	pubResp, _ := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "Test Song", Artist: "Test Artist",
		FileHash: "abc123", FileName: "test.mp3",
	})

	results, err := svc.CheckSongsByHash(ctx, []string{"abc123", "def456"})
	if err != nil {
		t.Fatalf("CheckSongsByHash() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if !results[0].Exists {
		t.Error("results[0].Exists = false for 'abc123', want true")
	}
	if results[0].Song.Id != pubResp.Id {
		t.Errorf("results[0].Song.Id = %q, want %q", results[0].Song.Id, pubResp.Id)
	}
	if results[1].Exists {
		t.Error("results[1].Exists = true for unknown hash, want false")
	}
}

func TestPublishSong_Success(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	resp, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "My Song", Artist: "Me", Album: "Album 1",
		FileHash: "def789", FileName: "track.flac",
	})
	if err != nil {
		t.Fatalf("PublishSong() error = %v", err)
	}
	if resp.Id == "" {
		t.Error("PublishSong() returned empty Id")
	}
	if resp.Title != "My Song" {
		t.Errorf("Title = %q, want %q", resp.Title, "My Song")
	}
}

func TestGetSong(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	pub, _ := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "Get Me", FileHash: "get123",
	})
	got, err := svc.GetSong(ctx, pub.Id)
	if err != nil {
		t.Fatalf("GetSong() error = %v", err)
	}
	if got.Title != "Get Me" {
		t.Errorf("Title = %q, want %q", got.Title, "Get Me")
	}
}

func TestSearchSongs(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Summer Nights", FileHash: "h1"})
	svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Winter Sun", FileHash: "h2"})
	svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Spring Rain", FileHash: "h3"})

	results, err := svc.SearchSongs(ctx, "summer", 10)
	if err != nil {
		t.Fatalf("SearchSongs() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchSongs('summer') = %d, want 1", len(results))
	}
}

func TestListSongs(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := song.NewService(client)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.PublishSong(ctx, &songpb.PublishSongRequest{
			Title: "Song", FileHash: "h",
		})
	}

	songs, err := svc.ListSongs(ctx, 3, 0)
	if err != nil {
		t.Fatalf("ListSongs() error = %v", err)
	}
	if len(songs) > 3 {
		t.Errorf("ListSongs(limit=3) = %d, want <=3", len(songs))
	}
}
