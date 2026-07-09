// Package e2e_test contains end-to-end integration tests for EchoVault.
package e2e_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newFullServer creates a test gRPC server with all services registered
// and auth interceptor enabled. Returns all service clients plus a cleanup function.
func newFullServer(t *testing.T) (
	userpb.UserServiceClient,
	songpb.SongServiceClient,
	playlistpb.PlaylistServiceClient,
	lyricpb.LyricServiceClient,
	func(),
) {
	t.Helper()

	name := "file:e2e_full_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	err = client.Schema.Create(context.Background())
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(evgrpc.AuthInterceptor(testJWTSecret)))
	evgrpc.RegisterAll(s, client, testJWTSecret)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	cleanup := func() { _ = conn.Close(); s.GracefulStop() }

	return userpb.NewUserServiceClient(conn),
		songpb.NewSongServiceClient(conn),
		playlistpb.NewPlaylistServiceClient(conn),
		lyricpb.NewLyricServiceClient(conn),
		cleanup
}

// registerAndLogin registers and logs in a user, returning the auth token.
func registerAndLogin(t *testing.T, userClient userpb.UserServiceClient, username, password string) string {
	t.Helper()

	_, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	loginResp, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	return loginResp.GetAccessToken()
}

// TestFullFlow_RegisterLoginPublishPlaylistLyric tests the complete user journey.
func TestFullFlow_RegisterLoginPublishPlaylistLyric(t *testing.T) {
	t.Parallel()
	userClient, songClient, playlistClient, lyricClient, cleanup := newFullServer(t)
	defer cleanup()

	// Step 1: Register
	_, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username:    "e2e_user",
		Password:    "SecurePass1",
		DisplayName: "E2E Tester",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Step 2: Login
	loginResp, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "e2e_user",
		Password: "SecurePass1",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	token := loginResp.GetAccessToken()
	if token == "" {
		t.Fatal("AccessToken is empty")
	}

	// Step 3: Publish songs (authenticated)
	ctx := authCtx(token)

	song1Resp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Test Song One",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Genre:    "Rock",
		FileHash: uuid.New().String(),
		FileName: "test1.mp3",
	})
	if err != nil {
		t.Fatalf("PublishSong 1 failed: %v", err)
	}
	song1ID := song1Resp.GetSong().GetId()
	if song1ID == "" {
		t.Fatal("Song 1 ID is empty")
	}

	_, err = songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Test Song Two",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Genre:    "Pop",
		FileHash: uuid.New().String(),
		FileName: "test2.mp3",
	})
	if err != nil {
		t.Fatalf("PublishSong 2 failed: %v", err)
	}

	// Step 4: Check songs by hash
	checkResp, err := songClient.CheckSongsByHash(ctx, &songpb.CheckSongsByHashRequest{
		FileHashes: []string{song1Resp.GetSong().GetFileHash(), "nonexistent-hash"},
	})
	if err != nil {
		t.Fatalf("CheckSongsByHash failed: %v", err)
	}
	if len(checkResp.GetResults()) != 2 {
		t.Fatalf("CheckSongsByHash returned %d results, want 2", len(checkResp.GetResults()))
	}
	if !checkResp.GetResults()[0].GetExists() {
		t.Error("First hash should exist")
	}
	if checkResp.GetResults()[1].GetExists() {
		t.Error("Second hash should not exist")
	}

	// Step 5: List songs
	listResp, err := songClient.ListSongs(ctx, &songpb.ListSongsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListSongs failed: %v", err)
	}
	if len(listResp.GetSongs()) != 2 {
		t.Fatalf("ListSongs returned %d songs, want 2", len(listResp.GetSongs()))
	}

	// Step 6: Search songs
	searchResp, err := songClient.SearchSongs(ctx, &songpb.SearchSongsRequest{
		Query: "Test Song One",
	})
	if err != nil {
		t.Fatalf("SearchSongs failed: %v", err)
	}
	if len(searchResp.GetSongs()) != 1 {
		t.Fatalf("SearchSongs returned %d songs, want 1", len(searchResp.GetSongs()))
	}

	// Step 7: Create playlist
	playlistResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name:        "My E2E Playlist",
		Description: "A test playlist from e2e test",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	playlistID := playlistResp.GetPlaylist().GetId()
	if playlistID == "" {
		t.Fatal("Playlist ID is empty")
	}

	// Step 8: Add songs to playlist
	_, err = playlistClient.AddSongToPlaylist(ctx, &playlistpb.AddSongToPlaylistRequest{
		PlaylistId: playlistID,
		SongId:     song1ID,
	})
	if err != nil {
		t.Fatalf("AddSongToPlaylist 1 failed: %v", err)
	}

	// Step 9: Save lyric
	_, err = lyricClient.SaveLyric(ctx, &lyricpb.SaveLyricRequest{
		SongId: song1ID,
		Type:   lyricpb.LyricType_LYRIC_TYPE_ORIGINAL,
		Content: `[00:01.00]This is a test lyric
[00:05.00]For e2e testing
[00:10.00]End of test`,
	})
	if err != nil {
		t.Fatalf("SaveLyric failed: %v", err)
	}

	// Step 10: Get lyric
	lyricResp, err := lyricClient.GetLyric(ctx, &lyricpb.GetLyricRequest{
		SongId: song1ID,
		Type:   lyricpb.LyricType_LYRIC_TYPE_ORIGINAL,
	})
	if err != nil {
		t.Fatalf("GetLyric failed: %v", err)
	}
	if lyricResp.GetLyric().GetContent() == "" {
		t.Error("Lyric content is empty")
	}

	// Step 11: Reorder playlist songs
	_, err = playlistClient.ReorderPlaylistSongs(ctx, &playlistpb.ReorderPlaylistSongsRequest{
		PlaylistId: playlistID,
		SongIds:    []string{song1ID},
	})
	if err != nil {
		t.Fatalf("ReorderPlaylistSongs failed: %v", err)
	}
}

// TestFullFlow_DuplicateRegistration tests that registering an existing user fails.
func TestFullFlow_DuplicateRegistration(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, cleanup := newFullServer(t)
	defer cleanup()

	_, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "duplicate_user",
		Password: "SecurePass1",
	})
	if err != nil {
		t.Fatalf("First register should succeed: %v", err)
	}

	_, err = userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "duplicate_user",
		Password: "SecurePass1",
	})
	if err == nil {
		t.Fatal("Duplicate register should fail")
	}
}

// TestFullFlow_RemoveSongFromPlaylist tests removing a song from a playlist.
func TestFullFlow_RemoveSongFromPlaylist(t *testing.T) {
	t.Parallel()
	userClient, songClient, playlistClient, _, cleanup := newFullServer(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "remove_test", "SecurePass1")
	ctx := authCtx(token)

	songResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Remove Me",
		Artist:   "Tester",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong: %v", err)
	}

	playlistResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name: "Remove Test Playlist",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}

	_, err = playlistClient.AddSongToPlaylist(ctx, &playlistpb.AddSongToPlaylistRequest{
		PlaylistId: playlistResp.GetPlaylist().GetId(),
		SongId:     songResp.GetSong().GetId(),
	})
	if err != nil {
		t.Fatalf("AddSongToPlaylist: %v", err)
	}

	_, err = playlistClient.RemoveSongFromPlaylist(ctx, &playlistpb.RemoveSongFromPlaylistRequest{
		PlaylistId: playlistResp.GetPlaylist().GetId(),
		SongId:     songResp.GetSong().GetId(),
	})
	if err != nil {
		t.Fatalf("RemoveSongFromPlaylist: %v", err)
	}
}

// TestFullFlow_UnauthenticatedAccess tests that protected RPCs reject unauthenticated calls.
func TestFullFlow_UnauthenticatedAccess(t *testing.T) {
	t.Parallel()
	_, songClient, _, _, cleanup := newFullServer(t)
	defer cleanup()

	_, err := songClient.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title:    "Should Fail",
		FileHash: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("Unauthenticated PublishSong should fail")
	}
}
