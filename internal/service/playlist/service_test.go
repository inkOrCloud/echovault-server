package playlist_test

import (
	"time"
	"errors"
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"testing"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	name := "file:pl_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestCreatePlaylist(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "My Playlist", "Description", "user-1")
	require.NoError(t, err)
	require.Equal(t, "My Playlist", p.Name)
	require.Equal(t, 0, p.SongCount)
}

func TestGetPlaylist(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "Get Me", "", "user-1")
	require.NoError(t, err)
	got, err := svc.GetPlaylist(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "Get Me", got.Name)
}

func TestUpdatePlaylist(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "Old Name", "", "user-1")
	require.NoError(t, err)
	updated, err := svc.UpdatePlaylist(ctx, p.ID, "New Name", "New Desc")
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.Name)
}

func TestDeletePlaylist(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "Delete Me", "", "user-1")
	require.NoError(t, err)
	require.NoError(t, svc.DeletePlaylist(ctx, p.ID))
	_, err = svc.GetPlaylist(ctx, p.ID)
	require.Error(t, err)
}

func TestListPlaylists(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	_, err := svc.CreatePlaylist(ctx, "P1", "", "user-1")
	require.NoError(t, err)
	_, err = svc.CreatePlaylist(ctx, "P2", "", "user-1")
	require.NoError(t, err)

	playlists, err := svc.ListPlaylists(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, playlists, 2)
}

func TestAddSongAndList(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "Songs", "", "user-1")
	require.NoError(t, err)
	ps, err := svc.AddSong(ctx, p.ID, "song-1", "user-1")
	require.NoError(t, err)
	require.Equal(t, "song-1", ps.SongID)

	songs, err := svc.ListPlaylistSongs(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, songs, 1)
}

func TestRemoveSong(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := playlist.NewService(client)
	ctx := context.Background()

	p, err := svc.CreatePlaylist(ctx, "Remove", "", "user-1")
	require.NoError(t, err)
	_, err = svc.AddSong(ctx, p.ID, "song-1", "user-1")
	require.NoError(t, err)
	require.NoError(t, svc.RemoveSong(ctx, p.ID, "song-1"))
	songs, err := svc.ListPlaylistSongs(ctx, p.ID)
	require.NoError(t, err)
	require.Empty(t, songs)
}

func TestAddSong_Duplicate(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client); ctx := context.Background()
	p, _ := svc.CreatePlaylist(ctx, "S", "", "u")
	svc.AddSong(ctx, p.ID, "s1", "u")
	_, err := svc.AddSong(ctx, p.ID, "s1", "u")
	if !errors.Is(err, playlist.ErrSongAlreadyInPlaylist) { t.Errorf("err=%v", err) }
}

func TestRemoveSong_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client); ctx := context.Background()
	p, _ := svc.CreatePlaylist(ctx, "R", "", "u")
	err := svc.RemoveSong(ctx, p.ID, "x")
	if !errors.Is(err, playlist.ErrSongNotFoundInPlaylist) { t.Errorf("err=%v", err) }
}

func TestGetPlaylist_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	_, err := svc.GetPlaylist(context.Background(), "x")
	if !errors.Is(err, playlist.ErrPlaylistNotFound) { t.Errorf("err=%v", err) }
}

func TestUpdatePlaylist_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	_, err := svc.UpdatePlaylist(context.Background(), "x", "N", "D")
	if !errors.Is(err, playlist.ErrPlaylistNotFound) { t.Errorf("err=%v", err) }
}

func TestReorderSongs(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client); ctx := context.Background()
	p, _ := svc.CreatePlaylist(ctx, "R", "", "u")
	for _, sid := range []string{"a","b","c"} { svc.AddSong(ctx, p.ID, sid, "u") }
	svc.ReorderSongs(ctx, p.ID, []string{"c","a","b"})
	songs, _ := svc.ListPlaylistSongs(ctx, p.ID)
	if songs[0].SongID != "c" || songs[2].SongID != "b" { t.Errorf("order wrong") }
}

func TestReorderSongs_Invalid(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client); ctx := context.Background()
	p, _ := svc.CreatePlaylist(ctx, "R", "", "u")
	svc.AddSong(ctx, p.ID, "s1", "u")
	if err := svc.ReorderSongs(ctx, p.ID, []string{"s1","x"}); err == nil { t.Fatal("expected error") }
}

func TestListPlaylists_Empty(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client)
	pl, _ := svc.ListPlaylists(context.Background(), "x")
	if len(pl) != 0 { t.Errorf("got %d", len(pl)) }
}

func TestAddSong_Position(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := playlist.NewService(client); ctx := context.Background()
	p, _ := svc.CreatePlaylist(ctx, "P", "", "u")
	ps1, _ := svc.AddSong(ctx, p.ID, "s1", "u")
	ps2, _ := svc.AddSong(ctx, p.ID, "s2", "u")
	if ps2.Position <= ps1.Position { t.Errorf("ps2=%d <= ps1=%d", ps2.Position, ps1.Position) }
}

func TestPlaylistEntToProto(t *testing.T) {
	t.Parallel()
	pb := playlist.EntToProto(&ent.Playlist{ID:"pl-1",Name:"Test",CreatedAt:time.Now(),UpdatedAt:time.Now()})
	if pb.GetId() != "pl-1" { t.Errorf("id=%q",pb.GetId()) }
	if pb.GetName() != "Test" { t.Errorf("name=%q",pb.GetName()) }
}

func TestPlaylistEntToProto_Nil(t *testing.T) {
	t.Parallel()
	if pb := playlist.EntToProto(nil); pb != nil { t.Error("should be nil") }
}
