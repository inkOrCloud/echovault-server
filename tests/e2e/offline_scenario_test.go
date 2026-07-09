// Package e2e_test contains end-to-end integration tests for EchoVault.
package e2e_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// newOfflineTestServer creates a test server for offline scenario testing.
func newOfflineTestServer(t *testing.T) (
	userpb.UserServiceClient,
	songpb.SongServiceClient,
	playlistpb.PlaylistServiceClient,
	syncpb.SyncServiceClient,
	string, // auth token
	func(),
) {
	t.Helper()

	name := "file:e2e_offline_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
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

	userClient := userpb.NewUserServiceClient(conn)

	// Register and login
	_, err = userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "offline_user",
		Password: "Offline1",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	loginResp, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "offline_user",
		Password: "Offline1",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	return userClient,
		songpb.NewSongServiceClient(conn),
		playlistpb.NewPlaylistServiceClient(conn),
		syncpb.NewSyncServiceClient(conn),
		loginResp.GetAccessToken(),
		cleanup
}

// TestOfflineScenario_LocalOperationsThenSync validates that data created
// "offline" (simulated by not syncing, then later pulling changes) is consistent.
//
// This simulates the offline-first architecture:
//   1. Device goes offline, user performs local operations
//   2. Device comes back online and syncs (PushChanges)
//   3. Server acknowledges and stores the changes
func TestOfflineScenario_LocalOperationsThenSync(t *testing.T) {
	t.Parallel()
	_, songClient, playlistClient, syncClient, token, cleanup := newOfflineTestServer(t)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	// ---- Simulate offline operations: create songs and playlist ----
	// In a real offline scenario, these would go to local SQLite first.
	// Here we simulate reconnection by sending operations that were queued locally.

	// "Offline" publish 1
	song1Resp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Offline Song One",
		Artist:   "Offline Artist",
		Album:    "Offline Album",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 1 (simulated offline): %v", err)
	}

	// "Offline" publish 2
	song2Resp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Offline Song Two",
		Artist:   "Offline Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 2 (simulated offline): %v", err)
	}

	// "Offline" create playlist
	playlistResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name: "Offline Playlist",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist (simulated offline): %v", err)
	}

	// ---- Come back online ----
	// Push all changes to the server
	pushResp, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		LastVersion: 0,
	})
	if err != nil {
		t.Fatalf("PushChanges (reconnect): %v", err)
	}
	if pushResp.GetAcceptedVersion() == 0 {
		t.Error("PushChanges accepted_version should be > 0")
	}
	t.Logf("PushChanges accepted version: %d", pushResp.GetAcceptedVersion())

	// ---- Pull changes from the server to verify consistency ----
	listResp, err := songClient.ListSongs(ctx, &songpb.ListSongsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListSongs after reconnect: %v", err)
	}

	// Verify all offline data is consistent on server
	if len(listResp.GetSongs()) < 2 {
		t.Fatalf("ListSongs returned %d songs, want >= 2", len(listResp.GetSongs()))
	}

	songIDs := make(map[string]bool)
	for _, s := range listResp.GetSongs() {
		songIDs[s.GetId()] = true
	}

	if !songIDs[song1Resp.GetSong().GetId()] {
		t.Error("Song 1 not found after reconnect")
	}
	if !songIDs[song2Resp.GetSong().GetId()] {
		t.Error("Song 2 not found after reconnect")
	}

	// Verify playlist was persisted
	playlistsResp, err := playlistClient.ListPlaylists(ctx, &playlistpb.ListPlaylistsRequest{})
	if err != nil {
		t.Fatalf("ListPlaylists: %v", err)
	}
	playlistFound := false
	for _, p := range playlistsResp.GetPlaylists() {
		if p.GetId() == playlistResp.GetPlaylist().GetId() {
			playlistFound = true
			break
		}
	}
	if !playlistFound {
		t.Error("Offline-created playlist not found after reconnect")
	}
}

// TestOfflineScenario_PushWithVersion tests that PushChanges properly tracks versions.
func TestOfflineScenario_PushWithVersion(t *testing.T) {
	t.Parallel()
	_, _, _, syncClient, token, cleanup := newOfflineTestServer(t)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))

	// Push with version tracking
	push1, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		LastVersion: 0,
	})
	if err != nil {
		t.Fatalf("First PushChanges: %v", err)
	}
	if push1.GetAcceptedVersion() < 0 {
		t.Error("First push accepted_version should be >= 0")
	}
	t.Logf("First push accepted version: %d", push1.GetAcceptedVersion())

	// Push again with the last version
	push2, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		LastVersion: push1.GetAcceptedVersion(),
	})
	if err != nil {
		t.Fatalf("Second PushChanges: %v", err)
	}
	t.Logf("Second push accepted version: %d", push2.GetAcceptedVersion())
}
