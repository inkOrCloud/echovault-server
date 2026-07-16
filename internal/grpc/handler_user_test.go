package grpc_test

import (
	metadata "google.golang.org/grpc/metadata"
	require "github.com/stretchr/testify/require"
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
)

const (
	testDeviceID = "dev-001"
	testPassword = "ValidPass1"
)

func newTestServer(t *testing.T) (userpb.UserServiceClient, func()) {
	t.Helper()
	name := "file:user_handler_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	err = client.Schema.Create(context.Background())
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	svc := user.NewService(client, "test-secret")
	handler := evgrpc.NewUserHandler(svc)
	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, handler)
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
	c := userpb.NewUserServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestUserRegisterHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newTestServer(t)
	defer cleanup()

	resp, err := c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "handler_user", Password: testPassword,
	})
	if err != nil {
		t.Fatalf("Register RPC error = %v", err)
	}
	if resp.GetUser().GetUsername() != "handler_user" {
		t.Errorf("Username = %q", resp.GetUser().GetUsername())
	}
	if resp.GetAccessToken() == "" {
		t.Error("AccessToken is empty")
	}
}

func TestUserLoginHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newTestServer(t)
	defer cleanup()

	_, _ = c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "login_handler", Password: testPassword,
	})

	resp, err := c.Login(context.Background(), &userpb.LoginRequest{
		Username: "login_handler", Password: testPassword,
	})
	if err != nil {
		t.Fatalf("Login RPC error = %v", err)
	}
	if resp.GetAccessToken() == "" {
		t.Error("AccessToken is empty")
	}
}

func newAuthUserTestServer(t *testing.T) (userpb.UserServiceClient, func()) {
	t.Helper()
	name := "file:user_authed_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	svc := user.NewService(client, "test-secret")
	handler := evgrpc.NewUserHandler(svc)
	s := grpc.NewServer(grpc.UnaryInterceptor(evgrpc.AuthInterceptor("test-secret")))
	userpb.RegisterUserServiceServer(s, handler)
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	c := userpb.NewUserServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func authCtx(token string) context.Context {
	return metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))
}

func TestUserGetCurrentUserHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newAuthUserTestServer(t)
	defer cleanup()

	reg, err := c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "getme_handler", Password: testPassword,
	})
	require.NoError(t, err)

	ctx := authCtx(reg.GetAccessToken())
	current, err := c.GetCurrentUser(ctx, &userpb.GetCurrentUserRequest{})
	require.NoError(t, err)
	require.Equal(t, "getme_handler", current.GetUser().GetUsername())
}

func TestUserListDevicesHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newAuthUserTestServer(t)
	defer cleanup()

	reg, err := c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "listdev_handler", Password: testPassword,
	})
	require.NoError(t, err)

	ctx := authCtx(reg.GetAccessToken())

	_, err = c.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId: "dev-handler-1", DeviceName: "Desktop", Platform: "linux",
	})
	require.NoError(t, err)

	devices, err := c.ListDevices(ctx, &userpb.ListDevicesRequest{})
	require.NoError(t, err)
	require.Len(t, devices.GetDevices(), 1)
	require.Equal(t, "dev-handler-1", devices.GetDevices()[0].GetDeviceId())
}

func TestUserRegisterAndRemoveDeviceHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newAuthUserTestServer(t)
	defer cleanup()

	reg, err := c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "rmdev_handler", Password: testPassword,
	})
	require.NoError(t, err)

	ctx := authCtx(reg.GetAccessToken())

	_, err = c.RegisterDevice(ctx, &userpb.RegisterDeviceRequest{
		DeviceId: "dev-to-remove", DeviceName: "Phone", Platform: "android",
	})
	require.NoError(t, err)

	_, err = c.RemoveDevice(ctx, &userpb.RemoveDeviceRequest{DeviceId: "dev-to-remove"})
	require.NoError(t, err)

	devices, err := c.ListDevices(ctx, &userpb.ListDevicesRequest{})
	require.NoError(t, err)
	require.Empty(t, devices.GetDevices())
}

func TestUserGetCurrentUser_Unauthenticated(t *testing.T) {
	t.Parallel()
	c, cleanup := newAuthUserTestServer(t)
	defer cleanup()

	_, err := c.GetCurrentUser(context.Background(), &userpb.GetCurrentUserRequest{})
	require.Error(t, err)
}

func TestUserRegisterDuplicateHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newAuthUserTestServer(t)
	defer cleanup()

	_, err := c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "dup_handler", Password: testPassword,
	})
	require.NoError(t, err)

	_, err = c.Register(context.Background(), &userpb.RegisterRequest{
		Username: "dup_handler", Password: testPassword + "2",
	})
	require.Error(t, err)
}
