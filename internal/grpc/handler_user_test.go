package grpc_test

import (
	"context"
	"net"
	"testing"

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
