package lyric_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	name := "file:lyric_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestSaveAndGetLyric(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := lyric.NewService(client)
	ctx := context.Background()

	lrc := "[00:01.00]Line 1\n[00:02.00]Line 2"
	saved, err := svc.SaveLyric(ctx, "song-1", lrc, lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)
	require.Equal(t, "song-1", saved.GetSongId())

	got, err := svc.GetLyric(ctx, "song-1", "", lyricpb.Lyric_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	require.Equal(t, lrc, got.GetContent())
}

func TestGetLyric_NotFound(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := lyric.NewService(client)
	_, err := svc.GetLyric(context.Background(), "no-such-song", "", lyricpb.Lyric_TYPE_UNSPECIFIED)
	require.Error(t, err)
}

func TestSaveLyric_UpdateExisting(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := lyric.NewService(client)
	ctx := context.Background()

	_, err := svc.SaveLyric(ctx, "song-1", "[00:01.00]v1", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)
	_, err = svc.SaveLyric(ctx, "song-1", "[00:01.00]v2", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)

	got, err := svc.GetLyric(ctx, "song-1", "zh", lyricpb.Lyric_TYPE_ORIGINAL)
	require.NoError(t, err)
	require.Equal(t, "[00:01.00]v2", got.GetContent())
}

func TestSaveLyric_MultipleLanguages(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := lyric.NewService(client)
	ctx := context.Background()

	_, err := svc.SaveLyric(ctx, "song-1", "zh lrc", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)
	_, err = svc.SaveLyric(ctx, "song-1", "en lrc", lyricpb.Lyric_TYPE_TRANSLATION, "en")
	require.NoError(t, err)

	lyrics, err := svc.GetAllLyrics(ctx, "song-1")
	require.NoError(t, err)
	require.Len(t, lyrics, 2)
}

func TestDeleteLyric(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := lyric.NewService(client)
	ctx := context.Background()

	_, err := svc.SaveLyric(ctx, "song-1", "lrc", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)
	err = svc.DeleteLyric(ctx, "song-1", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	require.NoError(t, err)
	_, err = svc.GetLyric(ctx, "song-1", "zh", lyricpb.Lyric_TYPE_ORIGINAL)
	require.Error(t, err)
}
