package lyric_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/lyric"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:lyric?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func TestSaveAndGetLyric(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	ctx := context.Background()

	lrc := "[00:01.00]Line 1\n[00:02.00]Line 2"
	saved, err := svc.SaveLyric(ctx, "song-1", lrc, lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	if err != nil { t.Fatalf("SaveLyric() error = %v", err) }
	if saved.SongId != "song-1" { t.Errorf("SongId = %q", saved.SongId) }

	got, err := svc.GetLyric(ctx, "song-1", "", lyricpb.Lyric_TYPE_UNSPECIFIED)
	if err != nil { t.Fatalf("GetLyric() error = %v", err) }
	if got.Content != lrc { t.Errorf("content mismatch") }
}

func TestGetLyric_NotFound(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	_, err := svc.GetLyric(context.Background(), "no-such-song", "", lyricpb.Lyric_TYPE_UNSPECIFIED)
	if err == nil { t.Fatal("expected error for nonexistent song") }
}

func TestSaveLyric_UpdateExisting(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	ctx := context.Background()

	svc.SaveLyric(ctx, "song-1", "[00:01.00]v1", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	svc.SaveLyric(ctx, "song-1", "[00:01.00]v2", lyricpb.Lyric_TYPE_ORIGINAL, "zh")

	got, _ := svc.GetLyric(ctx, "song-1", "zh", lyricpb.Lyric_TYPE_ORIGINAL)
	if got.Content != "[00:01.00]v2" { t.Errorf("content = %q, want v2", got.Content) }
}

func TestSaveLyric_MultipleLanguages(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	ctx := context.Background()

	svc.SaveLyric(ctx, "song-1", "zh lrc", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	svc.SaveLyric(ctx, "song-1", "en lrc", lyricpb.Lyric_TYPE_TRANSLATION, "en")

	lyrics, err := svc.GetAllLyrics(ctx, "song-1")
	if err != nil { t.Fatalf("GetAllLyrics() error = %v", err) }
	if len(lyrics) != 2 { t.Errorf("lyrics count = %d, want 2", len(lyrics)) }
}

func TestDeleteLyric(t *testing.T) {
	client := newTestClient(t); defer client.Close()
	svc := lyric.NewService(client)
	ctx := context.Background()

	svc.SaveLyric(ctx, "song-1", "lrc", lyricpb.Lyric_TYPE_ORIGINAL, "zh")
	if err := svc.DeleteLyric(ctx, "song-1", lyricpb.Lyric_TYPE_ORIGINAL, "zh"); err != nil {
		t.Fatalf("DeleteLyric() error = %v", err)
	}
	_, err := svc.GetLyric(ctx, "song-1", "zh", lyricpb.Lyric_TYPE_ORIGINAL)
	if err == nil { t.Fatal("expected error after delete") }
}
