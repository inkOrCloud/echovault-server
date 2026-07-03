package song_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	name := "file:song_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestCheckSongsByHash_Empty(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	results, err := svc.CheckSongsByHash(ctx, []string{"hash-nonexistent"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.False(t, results[0].Exists)
}

func TestCheckSongsByHash_Found(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	pubResp, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "Test Song", Artist: "Test Artist",
		FileHash: "abc123", FileName: "test.mp3",
	})
	require.NoError(t, err)

	results, err := svc.CheckSongsByHash(ctx, []string{"abc123", "def456"})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.True(t, results[0].Exists)
	require.Equal(t, pubResp.GetId(), results[0].Song.GetId())
	require.False(t, results[1].Exists)
}

func TestPublishSong_Success(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	resp, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "My Song", Artist: "Me", Album: "Album 1",
		FileHash: "def789", FileName: "track.flac",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetId())
	require.Equal(t, "My Song", resp.GetTitle())
}

func TestGetSong(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	pub, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "Get Me", FileHash: "get123",
	})
	require.NoError(t, err)
	got, err := svc.GetSong(ctx, pub.GetId())
	require.NoError(t, err)
	require.Equal(t, "Get Me", got.GetTitle())
}

func TestSearchSongs(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	_, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Summer Nights", FileHash: "h1"})
	require.NoError(t, err)
	_, err = svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Winter Sun", FileHash: "h2"})
	require.NoError(t, err)
	_, err = svc.PublishSong(ctx, &songpb.PublishSongRequest{Title: "Spring Rain", FileHash: "h3"})
	require.NoError(t, err)

	results, err := svc.SearchSongs(ctx, "summer", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestListSongs(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	for range 5 {
		_, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
			Title: "Song", FileHash: "h",
		})
		require.NoError(t, err)
	}

	songs, err := svc.ListSongs(ctx, 3, 0)
	require.NoError(t, err)
	require.LessOrEqual(t, len(songs), 3)
}

func TestUpdateFromScan_FillsEmptyFields(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	pubResp, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "User Title", Artist: "", Album: "",
		FileHash: "old-hash", FileName: "user.mp3",
	})
	require.NoError(t, err)

	err = svc.UpdateFromScan(ctx, pubResp.GetId(), "Tag Title", "Tag Artist", "Tag Album",
		"Tag Genre", 3, 1, 2024, "new-hash-abc", "song.mp3", "audio/mpeg", 12345)
	require.NoError(t, err)

	updated, err := svc.GetSong(ctx, pubResp.GetId())
	require.NoError(t, err)
	require.Equal(t, "User Title", updated.GetTitle())
	require.Equal(t, "Tag Artist", updated.GetArtist())
	require.Equal(t, "Tag Album", updated.GetAlbum())
	require.Equal(t, "new-hash-abc", updated.GetFileHash())
	require.Equal(t, int64(12345), updated.GetFileSize())
}

func TestUpdateFromScan_AllFieldsAlreadySet(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := song.NewService(client)
	ctx := context.Background()

	pubResp, err := svc.PublishSong(ctx, &songpb.PublishSongRequest{
		Title: "My Title", Artist: "My Artist", Album: "My Album",
		Genre: "Rock", FileHash: "hash1", FileName: "track.mp3",
	})
	require.NoError(t, err)

	err = svc.UpdateFromScan(ctx, pubResp.GetId(),
		"Tag Title", "Tag Artist", "Tag Album", "Tag Genre",
		3, 1, 2024, "new-hash", "track.mp3", "audio/mpeg", 9999)
	require.NoError(t, err)

	updated, err := svc.GetSong(ctx, pubResp.GetId())
	require.NoError(t, err)
	require.Equal(t, "My Title", updated.GetTitle())
	require.Equal(t, "My Artist", updated.GetArtist())
	require.Equal(t, "new-hash", updated.GetFileHash())
}
