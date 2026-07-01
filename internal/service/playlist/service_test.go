package playlist_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	drv, _ := entsql.Open("sqlite3", "file:playlist?mode=memory&cache=shared&_fk=1")
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func TestCreatePlaylist(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "My Playlist", "Description", "user-1")
	if err != nil { t.Fatalf("CreatePlaylist() error = %v", err) }
	if p.Name != "My Playlist" { t.Errorf("Name = %q", p.Name) }
	if p.SongCount != 0 { t.Errorf("SongCount = %d, want 0", p.SongCount) }
}

func TestGetPlaylist(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, _ := svc.CreatePlaylist(ctx, "Get Me", "", "user-1")
	got, err := svc.GetPlaylist(ctx, p.ID)
	if err != nil { t.Fatalf("GetPlaylist() error = %v", err) }
	if got.Name != "Get Me" { t.Errorf("Name = %q", got.Name) }
}

func TestUpdatePlaylist(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, _ := svc.CreatePlaylist(ctx, "Old Name", "", "user-1")
	updated, err := svc.UpdatePlaylist(ctx, p.ID, "New Name", "New Desc")
	if err != nil { t.Fatalf("UpdatePlaylist() error = %v", err) }
	if updated.Name != "New Name" { t.Errorf("Name = %q", updated.Name) }
}

func TestDeletePlaylist(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, _ := svc.CreatePlaylist(ctx, "Delete Me", "", "user-1")
	if err := svc.DeletePlaylist(ctx, p.ID); err != nil {
		t.Fatalf("DeletePlaylist() error = %v", err)
	}
	_, err := svc.GetPlaylist(ctx, p.ID)
	if err == nil { t.Fatal("expected error after delete") }
}

func TestListPlaylists(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	svc.CreatePlaylist(ctx, "P1", "", "user-1")
	svc.CreatePlaylist(ctx, "P2", "", "user-1")

	playlists, err := svc.ListPlaylists(ctx, "user-1")
	if err != nil { t.Fatalf("ListPlaylists() error = %v", err) }
	if len(playlists) != 2 { t.Errorf("count = %d, want 2", len(playlists)) }
}

func TestAddSongAndList(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, _ := svc.CreatePlaylist(ctx, "Songs", "", "user-1")
	ps, err := svc.AddSong(ctx, p.ID, "song-1", "user-1")
	if err != nil { t.Fatalf("AddSong() error = %v", err) }
	if ps.SongID != "song-1" { t.Errorf("SongId = %q", ps.SongID) }

	songs, err := svc.ListPlaylistSongs(ctx, p.ID)
	if err != nil { t.Fatalf("ListPlaylistSongs() error = %v", err) }
	if len(songs) != 1 { t.Errorf("count = %d, want 1", len(songs)) }
}

func TestRemoveSong(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, _ := svc.CreatePlaylist(ctx, "Remove", "", "user-1")
	svc.AddSong(ctx, p.ID, "song-1", "user-1")
	if err := svc.RemoveSong(ctx, p.ID, "song-1"); err != nil {
		t.Fatalf("RemoveSong() error = %v", err)
	}
	songs, _ := svc.ListPlaylistSongs(ctx, p.ID)
	if len(songs) != 0 { t.Errorf("count = %d, want 0", len(songs)) }
}
