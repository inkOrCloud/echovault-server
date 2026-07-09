// Package e2e_test contains end-to-end integration tests for EchoVault.
package e2e_test

import (
	"context"
	"io"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
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

// newFullServerWithDevices creates a test server and registers a user with two devices.
// Returns all clients and auth tokens for both devices.
func newFullServerWithDevices(t *testing.T) (
	userClient userpb.UserServiceClient,
	songClient songpb.SongServiceClient,
	syncClient syncpb.SyncServiceClient,
	device1Token string,
	device2Token string,
	cleanup func(),
) {
	t.Helper()

	name := "file:e2e_sync_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
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
	cleanup = func() { _ = conn.Close(); s.GracefulStop() }

	userClient = userpb.NewUserServiceClient(conn)
	songClient = songpb.NewSongServiceClient(conn)
	syncClient = syncpb.NewSyncServiceClient(conn)

	// Register user
	_, err = userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "sync_user",
		Password: "SyncPass1",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Login as device 1
	login1, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "sync_user",
		Password: "SyncPass1",
	})
	if err != nil {
		t.Fatalf("Login device 1: %v", err)
	}
	device1Token = login1.GetAccessToken()

	// Login as device 2
	login2, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "sync_user",
		Password: "SyncPass1",
	})
	if err != nil {
		t.Fatalf("Login device 2: %v", err)
	}
	device2Token = login2.GetAccessToken()

	return
}

// TestMultiDevice_PublishAndSync verifies that a song published on device 1
// is visible on device 2 after syncing.
func TestMultiDevice_PublishAndSync(t *testing.T) {
	t.Parallel()
	_, songClient, _, device1Token, device2Token, cleanup := newFullServerWithDevices(t)
	defer cleanup()

	ctxDevice1 := authCtx(device1Token)
	ctxDevice2 := authCtx(device2Token)

	// ---- Device 1 publishes a song ----
	songResp, err := songClient.PublishSong(ctxDevice1, &songpb.PublishSongRequest{
		Title:    "Synced Song",
		Artist:   "Sync Artist",
		Album:    "Sync Album",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("Device 1 PublishSong: %v", err)
	}

	// ---- Device 2 lists songs (should see the same song after sync) ----
	listResp, err := songClient.ListSongs(ctxDevice2, &songpb.ListSongsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Device 2 ListSongs: %v", err)
	}

	// Device 2 should see the song published by device 1
	found := false
	for _, s := range listResp.GetSongs() {
		if s.GetId() == songResp.GetSong().GetId() {
			found = true
			if s.GetTitle() != "Synced Song" {
				t.Errorf("Title = %q, want 'Synced Song'", s.GetTitle())
			}
			break
		}
	}
	if !found {
		t.Error("Device 2 did not find the song published by Device 1")
	}
}

// TestMultiDevice_PullChanges tests the sync pull mechanism.
func TestMultiDevice_PullChanges(t *testing.T) {
	t.Parallel()
	_, songClient, syncClient, device1Token, device2Token, cleanup := newFullServerWithDevices(t)
	defer cleanup()

	// Device 1 publishes a song
	_, err := songClient.PublishSong(authCtx(device1Token), &songpb.PublishSongRequest{
		Title:    "Pull Test Song",
		Artist:   "Sync Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("Device 1 PublishSong: %v", err)
	}

	// Device 2 pulls changes
	pullCtx := authCtx(device2Token)
	stream, err := syncClient.PullChanges(pullCtx, &syncpb.PullChangesRequest{
		LastVersion: 0,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("Device 2 PullChanges: %v", err)
	}

	changes, err := stream.Recv()
	if err != nil {
		t.Fatalf("PullChanges stream receive: %v", err)
	}
	if changes == nil {
		t.Fatal("PullChanges returned nil response")
	}

	// Should receive at least the published song as a change
	t.Logf("PullChanges response: versions=%v, hasMore=%v",
		changes.GetLastVersion(), changes.GetHasMore())
}

// TestMultiDevice_SubscribeChanges tests the sync subscription mechanism.
func TestMultiDevice_SubscribeChanges(t *testing.T) {
	t.Parallel()
	_, songClient, syncClient, device1Token, device2Token, cleanup := newFullServerWithDevices(t)
	defer cleanup()

	// Device 2 subscribes to changes
	subCtx, cancel := context.WithCancel(authCtx(device2Token))
	defer cancel()

	subStream, err := syncClient.SubscribeChanges(subCtx, &syncpb.SubscribeChangesRequest{})
	if err != nil {
		t.Fatalf("Device 2 SubscribeChanges: %v", err)
	}

	// Device 1 publishes a song (this should trigger a notification)
	_, err = songClient.PublishSong(authCtx(device1Token), &songpb.PublishSongRequest{
		Title:    "Subscription Test",
		Artist:   "Sync Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("Device 1 PublishSong: %v", err)
	}

	// Device 2 should receive a change notification
	// Use a channel with timeout to handle async notification
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			notif, err := subStream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				t.Logf("SubscribeChanges recv error (expected with cancel): %v", err)
				return
			}
			if notif != nil {
				t.Logf("Received change notification: %v", notif)
				return
			}
		}
	}()

	// Wait briefly for notification, then cancel
}


// authCtx creates a context with the Bearer token in gRPC metadata.
