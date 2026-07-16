package grpc_test

import (
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/playlist"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
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
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
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

func TestPlaylistAddSongAndListHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newPlaylistTestServer(t)
	defer cleanup()

	pl, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "Songs List",
	})
	require.NoError(t, err)

	_, err = c.AddSong(context.Background(), &playlistpb.AddSongRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
		SongId:     "song-1",
	})
	require.NoError(t, err)

	songs, err := c.ListPlaylistSongs(context.Background(), &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
	})
	require.NoError(t, err)
	require.Len(t, songs.GetSongs(), 1)
	require.Equal(t, "song-1", songs.GetSongs()[0].GetSongId())
}

func TestPlaylistUpdateAndDeleteHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newPlaylistTestServer(t)
	defer cleanup()

	pl, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "Original",
	})
	require.NoError(t, err)

	updated, err := c.UpdatePlaylist(context.Background(), &playlistpb.UpdatePlaylistRequest{
		Id:   pl.GetPlaylist().GetId(),
		Name: "Updated Name",
	})
	require.NoError(t, err)
	require.Equal(t, "Updated Name", updated.GetPlaylist().GetName())

	_, err = c.DeletePlaylist(context.Background(), &playlistpb.DeletePlaylistRequest{
		Id: pl.GetPlaylist().GetId(),
	})
	require.NoError(t, err)
}

func TestPlaylistRemoveSongHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newPlaylistTestServer(t)
	defer cleanup()

	pl, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "Remove Test",
	})
	require.NoError(t, err)

	_, err = c.AddSong(context.Background(), &playlistpb.AddSongRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
		SongId:     "song-to-remove",
	})
	require.NoError(t, err)

	_, err = c.RemoveSong(context.Background(), &playlistpb.RemoveSongRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
		SongId:     "song-to-remove",
	})
	require.NoError(t, err)

	songs, err := c.ListPlaylistSongs(context.Background(), &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
	})
	require.NoError(t, err)
	require.Empty(t, songs.GetSongs())
}

func TestPlaylistReorderSongsHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newPlaylistTestServer(t)
	defer cleanup()

	pl, err := c.CreatePlaylist(context.Background(), &playlistpb.CreatePlaylistRequest{
		Name: "Reorder",
	})
	require.NoError(t, err)

	for _, sid := range []string{"a", "b", "c"} {
		_, err = c.AddSong(context.Background(), &playlistpb.AddSongRequest{
			PlaylistId: pl.GetPlaylist().GetId(),
			SongId:     sid,
		})
		require.NoError(t, err)
	}

	_, err = c.ReorderSongs(context.Background(), &playlistpb.ReorderSongsRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
		SongIds:    []string{"c", "a", "b"},
	})
	require.NoError(t, err)

	songs, err := c.ListPlaylistSongs(context.Background(), &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: pl.GetPlaylist().GetId(),
	})
	require.NoError(t, err)
	require.Len(t, songs.GetSongs(), 3)
	require.Equal(t, "c", songs.GetSongs()[0].GetSongId())
}
