package playlist_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
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
