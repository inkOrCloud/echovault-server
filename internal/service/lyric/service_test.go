package lyric_test

import (
	"time"
	"errors"
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestSaveLyric_AllTypes(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client); ctx := context.Background()
	for _, typ := range []lyricpb.Lyric_Type{lyricpb.Lyric_TYPE_ORIGINAL,lyricpb.Lyric_TYPE_TRANSLATION,lyricpb.Lyric_TYPE_PHONETIC} {
		if _, err := svc.SaveLyric(ctx, "st", "l", typ, "en"); err != nil { t.Errorf("err=%v",err) }
	}
	lrcs, _ := svc.GetAllLyrics(ctx, "st")
	if len(lrcs) != 3 { t.Errorf("got %d", len(lrcs)) }
}

func TestDeleteLyric_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	err := svc.DeleteLyric(context.Background(), "x", lyricpb.Lyric_TYPE_ORIGINAL, "en")
	if !errors.Is(err, lyric.ErrLyricNotFound) { t.Errorf("err=%v", err) }
}

func TestGetAllLyrics_Empty(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	lrcs, _ := svc.GetAllLyrics(context.Background(), "x")
	if len(lrcs) != 0 { t.Errorf("got %d", len(lrcs)) }
}

func TestGetLyric_ByLanguageAndType(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client); ctx := context.Background()
	svc.SaveLyric(ctx,"sf","o",lyricpb.Lyric_TYPE_ORIGINAL,"en")
	svc.SaveLyric(ctx,"sf","t",lyricpb.Lyric_TYPE_TRANSLATION,"zh")
	got, _ := svc.GetLyric(ctx,"sf","zh",lyricpb.Lyric_TYPE_TRANSLATION)
	if got.GetContent() != "t" { t.Errorf("content=%q",got.GetContent()) }
}

func TestSaveLyric_UnspecifiedType(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client); ctx := context.Background()
	lrc, _ := svc.SaveLyric(ctx,"su","c",lyricpb.Lyric_TYPE_UNSPECIFIED,"en")
	if lrc.GetSongId() != "su" { t.Errorf("SongId=%q",lrc.GetSongId()) }
}

func TestLyricEntToProto(t *testing.T) {
	t.Parallel()
	pb := lyric.EntToProto(&ent.Lyric{SongID:"s1",Content:"hi",Type:"original",Language:"en",Source:"manual",UpdatedAt:time.Now()})
	if pb.GetType() != lyricpb.Lyric_TYPE_ORIGINAL { t.Errorf("Type=%v",pb.GetType()) }
	if pb.GetSource() != lyricpb.Lyric_SOURCE_MANUAL { t.Errorf("Source=%v",pb.GetSource()) }
}

func TestLyricEntToProto_Nil(t *testing.T) {
	t.Parallel()
	if pb := lyric.EntToProto(nil); pb != nil { t.Error("should be nil") }
}


func TestLyricEntToProto_AllTypes(t *testing.T) {
	t.Parallel(); now := time.Now()
	tests := []struct{name, entType string; want lyricpb.Lyric_Type}{
		{"original","original",lyricpb.Lyric_TYPE_ORIGINAL},
		{"translation","translation",lyricpb.Lyric_TYPE_TRANSLATION},
		{"phonetic","phonetic",lyricpb.Lyric_TYPE_PHONETIC},
		{"unknown","x",lyricpb.Lyric_TYPE_UNSPECIFIED},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pb := lyric.EntToProto(&ent.Lyric{SongID:"s",Content:"l",Type:tc.entType,Language:"e",UpdatedAt:now})
			if pb.GetType() != tc.want { t.Errorf("got %v, want %v", pb.GetType(), tc.want) }
		})
	}
}

func TestLyricEntToProto_AllSources(t *testing.T) {
	t.Parallel(); now := time.Now()
	tests := []struct{name, source string; want lyricpb.Lyric_Source}{
		{"embedded","embedded",lyricpb.Lyric_SOURCE_EMBEDDED},
		{"manual","manual",lyricpb.Lyric_SOURCE_MANUAL},
		{"fetched","fetched",lyricpb.Lyric_SOURCE_FETCHED},
		{"unknown","x",lyricpb.Lyric_SOURCE_UNSPECIFIED},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pb := lyric.EntToProto(&ent.Lyric{SongID:"s",Content:"l",Type:"original",Language:"e",Source:tc.source,UpdatedAt:now})
			if pb.GetSource() != tc.want { t.Errorf("got %v, want %v", pb.GetSource(), tc.want) }
		})
	}
}
