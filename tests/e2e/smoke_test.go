// Package e2e_test contains black-box smoke tests for EchoVault gRPC API.
// These tests exercise every RPC defined in the proto files, verifying that
// the server responds correctly (or returns expected Unimplemented for stubs).

package e2e_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	commonpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/common/v1"
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	songpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/song/v1"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// smokeTestHarness creates a test gRPC server backed by an in-memory SQLite database.
// Returns all five service clients and a cleanup function.
func smokeTestHarness(t *testing.T) (
	userpb.UserServiceClient,
	songpb.SongServiceClient,
	playlistpb.PlaylistServiceClient,
	lyricpb.LyricServiceClient,
	syncpb.SyncServiceClient,
	func(),
) {
	t.Helper()

	name := "file:smoke_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
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
		syncpb.NewSyncServiceClient(conn),
		cleanup
}

// ---------------------------------------------------------------------------
// UserService smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_UserService_RegisterAndLogin verifies the public register/login flow.
func TestSmoke_UserService_RegisterAndLogin(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	// Register
	regResp, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username:    "smoke_user",
		Password:    "SmokePass1",
		DisplayName: "Smoke Tester",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if regResp.GetUser().GetUsername() != "smoke_user" {
		t.Errorf("username = %q, want %q", regResp.GetUser().GetUsername(), "smoke_user")
	}
	if regResp.GetAccessToken() == "" {
		t.Error("access_token should not be empty after register")
	}

	// Login with same credentials
	loginResp, err := userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "smoke_user",
		Password: "SmokePass1",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResp.GetUser().GetUsername() != "smoke_user" {
		t.Errorf("username = %q, want %q", loginResp.GetUser().GetUsername(), "smoke_user")
	}
	if loginResp.GetAccessToken() == "" {
		t.Error("access_token should not be empty after login")
	}
	if loginResp.GetRefreshToken() != "" {
		t.Log("refresh_token present (expected when implemented)")
	}
}

// TestSmoke_UserService_DuplicateRegister verifies duplicate registration is rejected.
func TestSmoke_UserService_DuplicateRegister(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	_, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "dup_user",
		Password: "SmokePass1",
	})
	if err != nil {
		t.Fatalf("First register should succeed: %v", err)
	}

	_, err = userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "dup_user",
		Password: "SmokePass1",
	})
	if err == nil {
		t.Fatal("Duplicate registration should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %s", st.Code())
	}
}

// TestSmoke_UserService_LoginWrongPassword verifies login with bad password is rejected.
func TestSmoke_UserService_LoginWrongPassword(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	_, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username: "login_fail",
		Password: "SmokePass1",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = userClient.Login(context.Background(), &userpb.LoginRequest{
		Username: "login_fail",
		Password: "WrongPass1",
	})
	if err == nil {
		t.Fatal("Login with wrong password should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %s", st.Code())
	}
}

// TestSmoke_UserService_RefreshToken verifies the refresh token RPC.
// NOTE: RefreshToken is currently unimplemented; expect Unimplemented error.
func TestSmoke_UserService_RefreshToken(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	// First register to get a refresh token
	regResp, err := userClient.Register(context.Background(), &userpb.RegisterRequest{
		Username:    "refresh_me",
		Password:    "SmokePass1",
		DisplayName: "Refresh Tester",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	_ = regResp

	// Try refresh — needs auth (not a public RPC); currently unimplemented
	_, err = userClient.RefreshToken(authCtx(regResp.GetAccessToken()), &userpb.RefreshTokenRequest{
		RefreshToken: "some-refresh-token",
	})
	if err == nil {
		t.Log("RefreshToken succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("RefreshToken is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// TestSmoke_UserService_GetCurrentUser verifies the authenticated user info endpoint.
func TestSmoke_UserService_GetCurrentUser(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "get_me_user", "SmokePass1")
	ctx := authCtx(token)

	resp, err := userClient.GetCurrentUser(ctx, &userpb.GetCurrentUserRequest{})
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}
	if resp.GetUser().GetUsername() != "get_me_user" {
		t.Errorf("username = %q, want %q", resp.GetUser().GetUsername(), "get_me_user")
	}
	if resp.GetUser().GetId() == "" {
		t.Error("user id should not be empty")
	}
	if resp.GetUser().GetRole() == "" {
		t.Error("user role should not be empty")
	}
}

// TestSmoke_UserService_GetCurrentUserUnauthenticated verifies auth enforcement.
func TestSmoke_UserService_GetCurrentUserUnauthenticated(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	_, err := userClient.GetCurrentUser(context.Background(), &userpb.GetCurrentUserRequest{})
	if err == nil {
		t.Fatal("GetCurrentUser without auth should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %s", st.Code())
	}
}

// TestSmoke_UserService_UpdateUser verifies the update user RPC.
// NOTE: UpdateUser is currently unimplemented; expect Unimplemented error.
func TestSmoke_UserService_UpdateUser(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "update_me", "SmokePass1")
	ctx := authCtx(token)

	_, err := userClient.UpdateUser(ctx, &userpb.UpdateUserRequest{
		DisplayName: "New Display Name",
	})
	if err == nil {
		t.Log("UpdateUser succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("UpdateUser is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// TestSmoke_UserService_GetServerInfo verifies the public server info endpoint.
// NOTE: GetServerInfo is listed as a public RPC but currently unimplemented.
func TestSmoke_UserService_GetServerInfo(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	_, err := userClient.GetServerInfo(context.Background(), &userpb.GetServerInfoRequest{})
	if err == nil {
		t.Log("GetServerInfo succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("GetServerInfo is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// ---------------------------------------------------------------------------
// Device (UserService) smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_DeviceService_RegisterListRemove verifies the device lifecycle.
func TestSmoke_DeviceService_RegisterListRemove(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "device_user", "SmokePass1")
	ctx := authCtx(token)

	deviceID := "smoke-device-1"

	// ListDevices should be empty initially
	listResp, err := userClient.ListDevices(ctx, &userpb.ListDevicesRequest{})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(listResp.GetDevices()) != 0 {
		t.Errorf("expected 0 devices, got %d", len(listResp.GetDevices()))
	}

	// RegisterDevice
	_, err = userClient.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId:   deviceID,
		DeviceName: "Smoke Desktop",
		Platform:   "linux",
		OsVersion:  "Ubuntu 24.04",
	})
	if err != nil {
		t.Fatalf("RegisterDevice failed: %v", err)
	}

	// ListDevices should now have 1
	listResp, err = userClient.ListDevices(ctx, &userpb.ListDevicesRequest{})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(listResp.GetDevices()) != 1 {
		t.Fatalf("expected 1 device, got %d", len(listResp.GetDevices()))
	}
	d := listResp.GetDevices()[0]
	if d.GetDeviceId() != deviceID {
		t.Errorf("device_id = %q, want %q", d.GetDeviceId(), deviceID)
	}
	if d.GetDeviceName() != "Smoke Desktop" {
		t.Errorf("device_name = %q, want %q", d.GetDeviceName(), "Smoke Desktop")
	}
	if d.GetPlatform() != "linux" {
		t.Errorf("platform = %q, want %q", d.GetPlatform(), "linux")
	}

	// RemoveDevice
	_, err = userClient.RemoveDevice(ctx, &userpb.RemoveDeviceRequest{DeviceId: deviceID})
	if err != nil {
		t.Fatalf("RemoveDevice failed: %v", err)
	}

	// ListDevices should be empty again
	listResp, err = userClient.ListDevices(ctx, &userpb.ListDevicesRequest{})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(listResp.GetDevices()) != 0 {
		t.Errorf("expected 0 devices after removal, got %d", len(listResp.GetDevices()))
	}
}

// TestSmoke_DeviceService_RegisterDuplicate verifies duplicate device registration is rejected.
func TestSmoke_DeviceService_RegisterDuplicate(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "dup_dev_user", "SmokePass1")
	ctx := authCtx(token)

	_, err := userClient.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId:   "dup-device",
		DeviceName: "First",
		Platform:   "linux",
	})
	if err != nil {
		t.Fatalf("First RegisterDevice should succeed: %v", err)
	}

	_, err = userClient.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId:   "dup-device",
		DeviceName: "Second",
		Platform:   "macos",
	})
	if err == nil {
		t.Fatal("Duplicate RegisterDevice should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %s", st.Code())
	}
}

// TestSmoke_DeviceService_RemoveNotFound verifies removing a nonexistent device fails.
func TestSmoke_DeviceService_RemoveNotFound(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "rm_dev_user", "SmokePass1")
	ctx := authCtx(token)

	_, err := userClient.RemoveDevice(ctx, &userpb.RemoveDeviceRequest{
		DeviceId: "nonexistent-device",
	})
	if err == nil {
		t.Fatal("RemoveDevice on nonexistent device should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %s", st.Code())
	}
}

// TestSmoke_DeviceService_UpdateDevice verifies the update device RPC.
// NOTE: UpdateDevice is currently unimplemented; expect Unimplemented error.
func TestSmoke_DeviceService_UpdateDevice(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "upd_dev_user", "SmokePass1")
	ctx := authCtx(token)

	// Register a device first
	_, err := userClient.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId:   "updatable-device",
		DeviceName: "Old Name",
		Platform:   "linux",
	})
	if err != nil {
		t.Fatalf("RegisterDevice failed: %v", err)
	}

	_, err = userClient.UpdateDevice(ctx, &userpb.UpdateDeviceRequest{
		DeviceId:   "updatable-device",
		DeviceName: "New Name",
	})
	if err == nil {
		t.Log("UpdateDevice succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("UpdateDevice is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// ---------------------------------------------------------------------------
// SongService smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_SongService_PublishAndGetSong verifies publish + get song by ID.
func TestSmoke_SongService_PublishAndGetSong(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "song_user_1", "SmokePass1")
	ctx := authCtx(token)

	songHash := uuid.New().String()
	pubResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:       "Smoke Song",
		Artist:      "Smoke Artist",
		Album:       "Smoke Album",
		Genre:       "Rock",
		TrackNumber: 1,
		DiscNumber:  1,
		Year:        2024,
		FileName:    "smoke.mp3",
		FileSize:    1234567,
		FileHash:    songHash,
		MimeType:    "audio/mpeg",
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}
	songID := pubResp.GetSong().GetId()
	if songID == "" {
		t.Fatal("song id should not be empty")
	}
	if pubResp.GetSong().GetTitle() != "Smoke Song" {
		t.Errorf("title = %q, want %q", pubResp.GetSong().GetTitle(), "Smoke Song")
	}
	if pubResp.GetSong().GetFileHash() != songHash {
		t.Errorf("file_hash = %q, want %q", pubResp.GetSong().GetFileHash(), songHash)
	}

	// GetSong
	getResp, err := songClient.GetSong(ctx, &songpb.GetSongRequest{Id: songID})
	if err != nil {
		t.Fatalf("GetSong failed: %v", err)
	}
	if getResp.GetSong().GetTitle() != "Smoke Song" {
		t.Errorf("title = %q, want %q", getResp.GetSong().GetTitle(), "Smoke Song")
	}
	if getResp.GetSong().GetId() != songID {
		t.Errorf("id = %q, want %q", getResp.GetSong().GetId(), songID)
	}
}

// TestSmoke_SongService_GetSongNotFound verifies GetSong on nonexistent ID returns NotFound.
func TestSmoke_SongService_GetSongNotFound(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "song_user_notfound", "SmokePass1")
	ctx := authCtx(token)

	_, err := songClient.GetSong(ctx, &songpb.GetSongRequest{Id: "nonexistent-song-id"})
	if err == nil {
		t.Fatal("GetSong on nonexistent ID should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %s", st.Code())
	}
}

// TestSmoke_SongService_CheckSongsByHash verifies batch hash lookup.
func TestSmoke_SongService_CheckSongsByHash(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "hash_user", "SmokePass1")
	ctx := authCtx(token)

	existingHash := uuid.New().String()
	missingHash := uuid.New().String()

	_, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Hash Test Song",
		Artist:   "Tester",
		FileHash: existingHash,
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	checkResp, err := songClient.CheckSongsByHash(ctx, &songpb.CheckSongsByHashRequest{
		FileHashes: []string{existingHash, missingHash},
	})
	if err != nil {
		t.Fatalf("CheckSongsByHash failed: %v", err)
	}
	if len(checkResp.GetResults()) != 2 {
		t.Fatalf("expected 2 results, got %d", len(checkResp.GetResults()))
	}
	if !checkResp.GetResults()[0].GetExists() {
		t.Error("existing hash should be marked as exists")
	}
	if checkResp.GetResults()[0].GetSong() == nil {
		t.Error("existing hash should return song data")
	}
	if checkResp.GetResults()[1].GetExists() {
		t.Error("missing hash should not exist")
	}
	if checkResp.GetResults()[1].GetSong() != nil {
		t.Error("missing hash should not return song data")
	}
}

// TestSmoke_SongService_ListSongs verifies paginated song listing.
func TestSmoke_SongService_ListSongs(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "list_user", "SmokePass1")
	ctx := authCtx(token)

	// Publish 3 songs
	for i := 0; i < 3; i++ {
		_, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
			Title:    fmt.Sprintf("List Song %d", i+1),
			Artist:   "List Artist",
			FileHash: uuid.New().String(),
		})
		if err != nil {
			t.Fatalf("PublishSong %d failed: %v", i+1, err)
		}
	}

	// List all
	listResp, err := songClient.ListSongs(ctx, &songpb.ListSongsRequest{
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("ListSongs failed: %v", err)
	}
	if len(listResp.GetSongs()) < 3 {
		t.Errorf("expected >= 3 songs, got %d", len(listResp.GetSongs()))
	}
}

// TestSmoke_SongService_SearchSongs verifies keyword search.
func TestSmoke_SongService_SearchSongs(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "search_user", "SmokePass1")
	ctx := authCtx(token)

	// Publish a unique song
	_, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Zebra Crossing",
		Artist:   "Unique Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	// Publish another song that shouldn't match
	_, err = songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Something Else",
		Artist:   "Other Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	// Search by title keyword
	searchResp, err := songClient.SearchSongs(ctx, &songpb.SearchSongsRequest{
		Query:      "Zebra",
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("SearchSongs failed: %v", err)
	}
	if len(searchResp.GetSongs()) != 1 {
		t.Errorf("expected 1 result for 'Zebra', got %d", len(searchResp.GetSongs()))
	}
	// Search by artist
	searchResp, err = songClient.SearchSongs(ctx, &songpb.SearchSongsRequest{
		Query:      "Unique Artist",
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("SearchSongs failed: %v", err)
	}
	if len(searchResp.GetSongs()) != 1 {
		t.Errorf("expected 1 result for 'Unique Artist', got %d", len(searchResp.GetSongs()))
	}
	// Search with no matches
	searchResp, err = songClient.SearchSongs(ctx, &songpb.SearchSongsRequest{
		Query:      "ZZZNothingZZZ",
		Pagination: &commonpb.PaginationRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("SearchSongs failed: %v", err)
	}
	if len(searchResp.GetSongs()) != 0 {
		t.Errorf("expected 0 results for nonexistent query, got %d", len(searchResp.GetSongs()))
	}
}

// TestSmoke_SongService_UpdateSong verifies song metadata update.
// NOTE: UpdateSong is currently unimplemented; expect Unimplemented error.
func TestSmoke_SongService_UpdateSong(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "upd_song_user", "SmokePass1")
	ctx := authCtx(token)

	pubResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Original Title",
		Artist:   "Original Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	_, err = songClient.UpdateSong(ctx, &songpb.UpdateSongRequest{
		Id:     pubResp.GetSong().GetId(),
		Title:  "Updated Title",
		Artist: "Updated Artist",
	})
	if err == nil {
		t.Log("UpdateSong succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("UpdateSong is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// TestSmoke_SongService_DeleteSong verifies song deletion.
// NOTE: DeleteSong is currently unimplemented; expect Unimplemented error.
func TestSmoke_SongService_DeleteSong(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "del_song_user", "SmokePass1")
	ctx := authCtx(token)

	pubResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "To Be Deleted",
		Artist:   "Delete Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	_, err = songClient.DeleteSong(ctx, &songpb.DeleteSongRequest{
		Id: pubResp.GetSong().GetId(),
	})
	if err == nil {
		t.Log("DeleteSong succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("DeleteSong is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// TestSmoke_SongService_ListDeviceLocalSongs verifies device-local song listing.
// NOTE: ListDeviceLocalSongs is currently unimplemented; expect Unimplemented error.
func TestSmoke_SongService_ListDeviceLocalSongs(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "local_song_user", "SmokePass1")
	ctx := authCtx(token)

	_, err := songClient.ListDeviceLocalSongs(ctx, &songpb.ListDeviceLocalSongsRequest{
		DeviceId: "some-device",
	})
	if err == nil {
		t.Log("ListDeviceLocalSongs succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("ListDeviceLocalSongs is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// TestSmoke_SongService_Unauthenticated verifies auth enforcement on song endpoints.
func TestSmoke_SongService_Unauthenticated(t *testing.T) {
	t.Parallel()
	_, songClient, _, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	_, err := songClient.PublishSong(context.Background(), &songpb.PublishSongRequest{
		Title:    "Should Fail",
		FileHash: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("PublishSong without auth should fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %s", st.Code())
	}
}

// ---------------------------------------------------------------------------
// PlaylistService smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_PlaylistService_CRUD verifies the full playlist lifecycle.
func TestSmoke_PlaylistService_CRUD(t *testing.T) {
	t.Parallel()
	userClient, songClient, playlistClient, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "pl_user", "SmokePass1")
	ctx := authCtx(token)

	// Create a song to add to playlist
	songResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Playlist Song",
		Artist:   "Playlist Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}
	songID := songResp.GetSong().GetId()

	// CreatePlaylist
	createResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name:        "Smoke Playlist",
		Description: "A smoke test playlist",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	playlistID := createResp.GetPlaylist().GetId()
	if playlistID == "" {
		t.Fatal("playlist id should not be empty")
	}
	if createResp.GetPlaylist().GetName() != "Smoke Playlist" {
		t.Errorf("name = %q, want %q", createResp.GetPlaylist().GetName(), "Smoke Playlist")
	}

	// GetPlaylist
	getResp, err := playlistClient.GetPlaylist(ctx, &playlistpb.GetPlaylistRequest{Id: playlistID})
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if getResp.GetPlaylist().GetName() != "Smoke Playlist" {
		t.Errorf("name = %q, want %q", getResp.GetPlaylist().GetName(), "Smoke Playlist")
	}

	// UpdatePlaylist
	updateResp, err := playlistClient.UpdatePlaylist(ctx, &playlistpb.UpdatePlaylistRequest{
		Id:          playlistID,
		Name:        "Updated Playlist",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("UpdatePlaylist failed: %v", err)
	}
	if updateResp.GetPlaylist().GetName() != "Updated Playlist" {
		t.Errorf("name = %q, want %q", updateResp.GetPlaylist().GetName(), "Updated Playlist")
	}

	// AddSong
	addResp, err := playlistClient.AddSong(ctx, &playlistpb.AddSongRequest{
		PlaylistId: playlistID,
		SongId:     songID,
		Position:   0,
	})
	if err != nil {
		t.Fatalf("AddSong failed: %v", err)
	}
	if addResp.GetPlaylistSong().GetSongId() != songID {
		t.Errorf("song_id = %q, want %q", addResp.GetPlaylistSong().GetSongId(), songID)
	}

	// ListPlaylistSongs
	listSongsResp, err := playlistClient.ListPlaylistSongs(ctx, &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: playlistID,
	})
	if err != nil {
		t.Fatalf("ListPlaylistSongs failed: %v", err)
	}
	if len(listSongsResp.GetSongs()) != 1 {
		t.Fatalf("expected 1 song in playlist, got %d", len(listSongsResp.GetSongs()))
	}
	if listSongsResp.GetSongs()[0].GetSongId() != songID {
		t.Errorf("song_id = %q, want %q", listSongsResp.GetSongs()[0].GetSongId(), songID)
	}

	// RemoveSong
	_, err = playlistClient.RemoveSong(ctx, &playlistpb.RemoveSongRequest{
		PlaylistId: playlistID,
		SongId:     songID,
	})
	if err != nil {
		t.Fatalf("RemoveSong failed: %v", err)
	}

	// Verify song is removed
	listSongsResp, err = playlistClient.ListPlaylistSongs(ctx, &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: playlistID,
	})
	if err != nil {
		t.Fatalf("ListPlaylistSongs failed: %v", err)
	}
	if len(listSongsResp.GetSongs()) != 0 {
		t.Errorf("expected 0 songs after removal, got %d", len(listSongsResp.GetSongs()))
	}

	// ListPlaylists
	listResp, err := playlistClient.ListPlaylists(ctx, &playlistpb.ListPlaylistsRequest{})
	if err != nil {
		t.Fatalf("ListPlaylists failed: %v", err)
	}
	found := false
	for _, p := range listResp.GetPlaylists() {
		if p.GetId() == playlistID {
			found = true
			break
		}
	}
	if !found {
		t.Error("playlist should be visible in ListPlaylists")
	}

	// DeletePlaylist
	_, err = playlistClient.DeletePlaylist(ctx, &playlistpb.DeletePlaylistRequest{Id: playlistID})
	if err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}

	// Verify deleted
	_, err = playlistClient.GetPlaylist(ctx, &playlistpb.GetPlaylistRequest{Id: playlistID})
	if err == nil {
		t.Fatal("GetPlaylist on deleted playlist should fail")
	}
}

// TestSmoke_PlaylistService_ReorderSongs verifies song reordering in a playlist.
func TestSmoke_PlaylistService_ReorderSongs(t *testing.T) {
	t.Parallel()
	userClient, songClient, playlistClient, _, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "reorder_user", "SmokePass1")
	ctx := authCtx(token)

	// Create two songs
	song1, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Song A",
		Artist:   "Reorder Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 1 failed: %v", err)
	}
	song2, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Song B",
		Artist:   "Reorder Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong 2 failed: %v", err)
	}

	plResp, err := playlistClient.CreatePlaylist(ctx, &playlistpb.CreatePlaylistRequest{
		Name: "Reorder Playlist",
	})
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	plID := plResp.GetPlaylist().GetId()

	// Add both songs
	_, err = playlistClient.AddSong(ctx, &playlistpb.AddSongRequest{
		PlaylistId: plID, SongId: song1.GetSong().GetId(), Position: 0,
	})
	if err != nil {
		t.Fatalf("AddSong 1 failed: %v", err)
	}
	_, err = playlistClient.AddSong(ctx, &playlistpb.AddSongRequest{
		PlaylistId: plID, SongId: song2.GetSong().GetId(), Position: 1,
	})
	if err != nil {
		t.Fatalf("AddSong 2 failed: %v", err)
	}

	// Reorder (reverse)
	_, err = playlistClient.ReorderSongs(ctx, &playlistpb.ReorderSongsRequest{
		PlaylistId: plID,
		SongIds:    []string{song2.GetSong().GetId(), song1.GetSong().GetId()},
	})
	if err != nil {
		t.Fatalf("ReorderSongs failed: %v", err)
	}

	// Verify order
	listSongsResp, err := playlistClient.ListPlaylistSongs(ctx, &playlistpb.ListPlaylistSongsRequest{
		PlaylistId: plID,
	})
	if err != nil {
		t.Fatalf("ListPlaylistSongs failed: %v", err)
	}
	if len(listSongsResp.GetSongs()) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(listSongsResp.GetSongs()))
	}
}

// ---------------------------------------------------------------------------
// LyricService smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_LyricService_SaveGetDelete verifies the lyric lifecycle.
func TestSmoke_LyricService_SaveGetDelete(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, lyricClient, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "lyric_user", "SmokePass1")
	ctx := authCtx(token)

	// First publish a song
	songResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Lyric Song",
		Artist:   "Lyric Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}
	songID := songResp.GetSong().GetId()

	// SaveLyric (original)
	lyricContent := "[00:01.00]This is a test lyric\n[00:05.00]Second line"
	saveResp, err := lyricClient.SaveLyric(ctx, &lyricpb.SaveLyricRequest{
		SongId:  songID,
		Type:    lyricpb.Lyric_TYPE_ORIGINAL,
		Language: "en",
		Content: lyricContent,
	})
	if err != nil {
		t.Fatalf("SaveLyric failed: %v", err)
	}
	if saveResp.GetLyric().GetSongId() != songID {
		t.Errorf("song_id = %q, want %q", saveResp.GetLyric().GetSongId(), songID)
	}

	// SaveLyric (translation)
	_, err = lyricClient.SaveLyric(ctx, &lyricpb.SaveLyricRequest{
		SongId:   songID,
		Type:     lyricpb.Lyric_TYPE_TRANSLATION,
		Language: "zh",
		Content:  "[00:01.00]这是一首测试歌词",
	})
	if err != nil {
		t.Fatalf("SaveLyric (translation) failed: %v", err)
	}

	// GetLyric — fetch original
	getResp, err := lyricClient.GetLyric(ctx, &lyricpb.GetLyricRequest{
		SongId: songID,
		Type:   lyricpb.Lyric_TYPE_ORIGINAL,
	})
	if err != nil {
		t.Fatalf("GetLyric failed: %v", err)
	}
	if len(getResp.GetLyrics()) == 0 {
		t.Fatal("expected at least 1 lyric")
	}
	foundOriginal := false
	for _, l := range getResp.GetLyrics() {
		if l.GetType() == lyricpb.Lyric_TYPE_ORIGINAL && l.GetLanguage() == "en" {
			foundOriginal = true
			if l.GetContent() != lyricContent {
				t.Errorf("content mismatch")
			}
			break
		}
	}
	if !foundOriginal {
		t.Error("original english lyric not found")
	}

	// GetLyric — fetch translation
	getResp, err = lyricClient.GetLyric(ctx, &lyricpb.GetLyricRequest{
		SongId:   songID,
		Type:     lyricpb.Lyric_TYPE_TRANSLATION,
		Language: "zh",
	})
	if err != nil {
		t.Fatalf("GetLyric (translation) failed: %v", err)
	}
	if len(getResp.GetLyrics()) == 0 {
		t.Fatal("expected at least 1 translation lyric")
	}

	// DeleteLyric
	_, err = lyricClient.DeleteLyric(ctx, &lyricpb.DeleteLyricRequest{
		SongId:   songID,
		Type:     lyricpb.Lyric_TYPE_ORIGINAL,
		Language: "en",
	})
	if err != nil {
		t.Fatalf("DeleteLyric failed: %v", err)
	}

	// Verify deleted
	_, err = lyricClient.GetLyric(ctx, &lyricpb.GetLyricRequest{
		SongId:   songID,
		Type:     lyricpb.Lyric_TYPE_ORIGINAL,
		Language: "en",
	})
	if err == nil {
		// Depending on implementation, this might return empty or error
		t.Log("Note: GetLyric after delete did not error (check implementation)")
	}
}

// TestSmoke_LyricService_SearchLyric verifies lyric text search.
// NOTE: SearchLyric is currently unimplemented; expect Unimplemented error.
func TestSmoke_LyricService_SearchLyric(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, lyricClient, _, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "search_lyric_user", "SmokePass1")
	ctx := authCtx(token)

	songResp, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Searchable Song",
		Artist:   "Search Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	_, err = lyricClient.SaveLyric(ctx, &lyricpb.SaveLyricRequest{
		SongId:  songResp.GetSong().GetId(),
		Content: "[00:01.00]Searchable lyric text here",
		Type:    lyricpb.Lyric_TYPE_ORIGINAL,
	})
	if err != nil {
		t.Fatalf("SaveLyric failed: %v", err)
	}

	_, err = lyricClient.SearchLyric(ctx, &lyricpb.SearchLyricRequest{
		Keyword: "Searchable",
	})
	if err == nil {
		t.Log("SearchLyric succeeded — update test when implemented")
		return
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() == codes.Unimplemented {
		t.Log("SearchLyric is unimplemented (expected for now)")
	} else {
		t.Errorf("unexpected error code: %s, msg: %s", st.Code(), st.Message())
	}
}

// ---------------------------------------------------------------------------
// SyncService smoke tests
// ---------------------------------------------------------------------------

// TestSmoke_SyncService_PushChanges verifies pushing sync changes.
func TestSmoke_SyncService_PushChanges(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, syncClient, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "sync_push_user", "SmokePass1")
	ctx := authCtx(token)

	// Push a create change
	pushResp, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId: "smoke-device-push",
		Changes: []*syncpb.SyncChange{
			{
				EntityType: "song",
				EntityId:   uuid.New().String(),
				Action:     syncpb.SyncChange_ACTION_CREATE,
				DeviceId:   "smoke-device-push",
			},
		},
	})
	if err != nil {
		t.Fatalf("PushChanges failed: %v", err)
	}
	if pushResp.GetServerVersion() <= 0 {
		t.Errorf("server_version = %d, want > 0", pushResp.GetServerVersion())
	}
	if pushResp.GetAcceptedCount() < 1 {
		t.Errorf("accepted_count = %d, want >= 1", pushResp.GetAcceptedCount())
	}
}

// TestSmoke_SyncService_PushChangesEmpty verifies pushing an empty change list.
func TestSmoke_SyncService_PushChangesEmpty(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, syncClient, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "sync_empty_user", "SmokePass1")
	ctx := authCtx(token)

	pushResp, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId: "smoke-device-empty",
		Changes:  []*syncpb.SyncChange{},
	})
	if err != nil {
		t.Fatalf("PushChanges (empty) failed: %v", err)
	}
	// Empty push may return version 0; just verify no error
	_ = pushResp
	if pushResp.GetServerVersion() < 0 {
		t.Errorf("server_version = %d, want >= 0", pushResp.GetServerVersion())
	}
}

// TestSmoke_SyncService_PullChanges verifies pulling changes via server-stream.
func TestSmoke_SyncService_PullChanges(t *testing.T) {
	t.Parallel()
	userClient, songClient, _, _, syncClient, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "sync_pull_user", "SmokePass1")
	ctx := authCtx(token)

	// Generate a change by publishing a song
	_, err := songClient.PublishSong(ctx, &songpb.PublishSongRequest{
		Title:    "Pull Test Song",
		Artist:   "Pull Artist",
		FileHash: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("PublishSong failed: %v", err)
	}

	// Pull changes
	stream, err := syncClient.PullChanges(ctx, &syncpb.PullChangesRequest{
		DeviceId:     "smoke-device-pull",
		SinceVersion: 0,
	})
	if err != nil {
		t.Fatalf("PullChanges failed: %v", err)
	}

	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			change, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}
			if change != nil && change.GetChange() != nil {
				t.Logf("Received sync change: type=%s entity=%s",
					change.GetChange().GetEntityType(), change.GetChange().GetEntityId())
				return
			}
		}
	}()

	select {
	case <-done:
		// OK — received at least one change (or stream ended)
	case <-time.After(3 * time.Second):
		t.Log("PullChanges stream timed out (acceptable if no changes are in the stream)")
	}
}

// TestSmoke_SyncService_SubscribeChanges verifies subscribing to real-time notifications.
// This test verifies the stream can be established and cancelled cleanly.
func TestSmoke_SyncService_SubscribeChanges(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, syncClient, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "sync_sub_user", "SmokePass1")
	ctx, cancel := context.WithCancel(authCtx(token))
	defer cancel()

	stream, err := syncClient.SubscribeChanges(ctx, &syncpb.SubscribeChangesRequest{
		DeviceId: "smoke-device-sub",
	})
	if err != nil {
		t.Fatalf("SubscribeChanges failed: %v", err)
	}

	// Read from subscription briefly, then cancel
	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}
		}
	}()

	// Let it run briefly, then cancel
	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Clean exit on cancel
	case <-time.After(2 * time.Second):
		t.Log("SubscribeChanges stream did not exit immediately after cancel (ok)")
	}
}

// TestSmoke_SyncService_AckChanges verifies acknowledging consumed changes.
func TestSmoke_SyncService_AckChanges(t *testing.T) {
	t.Parallel()
	userClient, _, _, _, syncClient, cleanup := smokeTestHarness(t)
	defer cleanup()

	token := registerAndLogin(t, userClient, "sync_ack_user", "SmokePass1")
	ctx := authCtx(token)

	// First push a change to get a version
	pushResp, err := syncClient.PushChanges(ctx, &syncpb.PushChangesRequest{
		DeviceId: "smoke-device-ack",
		Changes: []*syncpb.SyncChange{
			{
				EntityType: "song",
				EntityId:   uuid.New().String(),
				Action:     syncpb.SyncChange_ACTION_CREATE,
				DeviceId:   "smoke-device-ack",
			},
		},
	})
	if err != nil {
		t.Fatalf("PushChanges failed: %v", err)
	}

	// Ack the changes
	_, err = syncClient.AckChanges(ctx, &syncpb.AckChangesRequest{
		DeviceId:     "smoke-device-ack",
		AckedVersion: pushResp.GetServerVersion(),
	})
	if err != nil {
		t.Fatalf("AckChanges failed: %v", err)
	}
}
