package e2e_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	commonpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/common/v1"
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
)

const offlineDeviceID = "e2e-offline-device"

// newOfflineTestServer creates a test server for offline scenario testing.
func newOfflineTestServer(t *testing.T) (
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

	return songpb.NewSongServiceClient(conn),
		playlistpb.NewPlaylistServiceClient(conn),
		syncpb.NewSyncServiceClient(conn),
		loginResp.GetAccessToken(),
		cleanup
}

// TestOfflineScenario_LocalOperationsThenSync validates that data created
// while "offline" is consistent after syncing.
func TestOfflineScenario_LocalOperationsThenSync(t *testing.T) {
	t.Parallel()
	songClient, playlistClient, syncClient, token, cleanup := newOfflineTestServer(t)
	defer cleanup()

	ctx := authCtx(token)

	// Simulate "offline" operations: create songs and playlist
	song1Resp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Offline Song One",
		Artist:   "Offline Artist",
		Album:    "Offline Album",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 1: %v", err)
	}

	song2Resp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Offline Song Two",
		Artist:   "Offline Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 2: %v", err)
	}

	playlistResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name: "Offline Playlist",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}

	// "Reconnect": push offline changes to server
	pushResp, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId: offlineDeviceID,
		Changes: []*syncpb.SyncChange{
			{
				EntityType: "song", //nolint:goconst
				EntityId:   song1Resp.GetSong().GetId(),
				Action:     syncpb.SyncChange_ACTION_CREATE,
				DeviceId:   offlineDeviceID,
			},
			{
				EntityType: "song",
				EntityId:   song2Resp.GetSong().GetId(),
				Action:     syncpb.SyncChange_ACTION_CREATE,
				DeviceId:   offlineDeviceID,
			},
		},
	})
	if err != nil {
		t.Fatalf("PushChanges: %v", err)
	}
	if pushResp.GetServerVersion() <= 0 {
		t.Error("PushChanges ServerVersion should be > 0")
	}
	t.Logf("PushChanges server version: %d, accepted: %d",
		pushResp.GetServerVersion(), pushResp.GetAcceptedCount())

	// Verify all offline data is consistent on server
	listResp, err := songClient.ListSongs(ctx, &songpb.ListSongsRequest{
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("ListSongs: %v", err)
	}
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
	found := false
	for _, p := range playlistsResp.GetPlaylists() {
		if p.GetId() == playlistResp.GetPlaylist().GetId() {
			found = true
			break
		}
	}
	if !found {
		t.Error("Offline-created playlist not found after reconnect")
	}
}

// TestOfflineScenario_PushWithVersion verifies push version tracking.
func TestOfflineScenario_PushWithVersion(t *testing.T) {
	t.Parallel()
	_, _, syncClient, token, cleanup := newOfflineTestServer(t)
	defer cleanup()

	ctx := authCtx(token)

	// Push with version tracking
	push1, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId: offlineDeviceID,
		Changes:  []*syncpb.SyncChange{},
	})
	if err != nil {
		t.Fatalf("First PushChanges: %v", err)
	}
	t.Logf("First push: version=%d accepted=%d",
		push1.GetServerVersion(), push1.GetAcceptedCount())

	// Push again — should work with any version
	push2, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId:        offlineDeviceID,
		LastPullVersion: push1.GetServerVersion(),
		Changes:         []*syncpb.SyncChange{},
	})
	if err != nil {
		t.Fatalf("Second PushChanges: %v", err)
	}
	t.Logf("Second push: version=%d accepted=%d",
		push2.GetServerVersion(), push2.GetAcceptedCount())
}
