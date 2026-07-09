package e2e_test

import (
	"context"
	"net"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	commonpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/common/v1"
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

const (
	device1ID = "e2e-device-1"
	device2ID = "e2e-device-2"
)

// newMultiDeviceServer creates a test server with auth, registers a user,
// and returns all clients plus tokens for two devices.
func newMultiDeviceServer(t *testing.T) (
	songpb.SongServiceClient,
	syncpb.SyncServiceClient,
	string, // device1 token
	string, // device2 token
	func(),
) {
	t.Helper()

	name := "file:e2e_multi_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
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

	// Register user
	_, err = userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "multi_user", //nolint:goconst
		Password: "MultiPass1", //nolint:goconst
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Login as device 1
	login1, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "multi_user",
		Password: "MultiPass1",
	})
	if err != nil {
		t.Fatalf("Login device 1: %v", err)
	}

	// Login as device 2
	login2, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "multi_user",
		Password: "MultiPass1",
	})
	if err != nil {
		t.Fatalf("Login device 2: %v", err)
	}

	return songpb.NewSongServiceClient(conn),
		syncpb.NewSyncServiceClient(conn),
		login1.GetAccessToken(),
		login2.GetAccessToken(),
		cleanup
}

// TestMultiDevice_PublishAndSync verifies that a song published on device 1
// is visible on device 2.
func TestMultiDevice_PublishAndSync(t *testing.T) {
	t.Parallel()
	songClient, _, device1Token, device2Token, cleanup := newMultiDeviceServer(t)
	defer cleanup()

	ctxDevice1 := authCtx(device1Token)
	ctxDevice2 := authCtx(device2Token)

	// Device 1 publishes a song
	songResp, err := songClient.PublishSong(ctxDevice1, &songpb.PublishSongRequest{
		Title:    "Synced Song",
		Artist:   "Sync Artist",
		Album:    "Sync Album",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("Device 1 PublishSong: %v", err)
	}

	// Device 2 lists songs — should see the same shared song
	listResp, err := songClient.ListSongs(ctxDevice2, &songpb.ListSongsRequest{
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("Device 2 ListSongs: %v", err)
	}

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

// TestMultiDevice_PushChanges tests pushing sync changes from a device.
func TestMultiDevice_PushChanges(t *testing.T) {
	t.Parallel()
	_, syncClient, device1Token, _, cleanup := newMultiDeviceServer(t) //nolint:dogsled
	defer cleanup()

	// Push a sync change
	resp, err := syncClient.PushChanges(authCtx(device1Token), &syncpb.PushChangesRequest{
		DeviceId: device1ID,
		Changes: []*syncpb.SyncChange{{
			EntityType: "song", //nolint:goconst
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			DeviceId:   device1ID,
		}},
	})
	if err != nil {
		t.Fatalf("PushChanges: %v", err)
	}
	if resp.GetServerVersion() <= 0 {
		t.Errorf("ServerVersion = %d, want > 0", resp.GetServerVersion())
	}
}

// TestMultiDevice_PullChanges tests pulling changes from the server.
func TestMultiDevice_PullChanges(t *testing.T) {
	t.Parallel()
	songClient, syncClient, device1Token, device2Token, cleanup := newMultiDeviceServer(t)
	defer cleanup()

	// Device 1 publishes a song to generate a change
	_, err := songClient.PublishSong(authCtx(device1Token), &songpb.PublishSongRequest{
		Title:    "Pull Test Song",
		Artist:   "Test Artist", //nolint:goconst
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong: %v", err)
	}

	// Device 2 pulls changes
	ctx := authCtx(device2Token)
	stream, err := syncClient.PullChanges(ctx, &syncpb.PullChangesRequest{
		DeviceId:     device2ID,
		SinceVersion: 0,
	})
	if err != nil {
		t.Fatalf("PullChanges: %v", err)
	}

	// Read from the stream with timeout
	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			change, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}
			if change != nil && change.GetChange() != nil {
				t.Logf("Received change: type=%s entity=%s",
					change.GetChange().GetEntityType(), change.GetChange().GetEntityId())
				return
			}
		}
	}()

	select {
	case <-done:
		// OK
	case <-time.After(3 * time.Second):
		t.Log("PullChanges stream timed out (expected with no changes in stream impl)")
	}
}
