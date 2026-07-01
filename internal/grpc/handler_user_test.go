package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	userpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/user/v1"
)

func newTestServer(t *testing.T) (userpb.UserServiceClient, func()) {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:handler?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	svc := user.NewService(client, "test-secret")
	userHandler := evgrpc.NewUserHandler(svc)

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, userHandler)

	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go s.Serve(lis)

	conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	uclient := userpb.NewUserServiceClient(conn)

	return uclient, func() {
		conn.Close()
		s.GracefulStop()
	}
}

func TestRegisterHandler_Success(t *testing.T) {
	client, cleanup := newTestServer(t)
	defer cleanup()

	resp, err := client.Register(context.Background(), &userpb.RegisterRequest{
		Username: "handleruser",
		Password: "ValidPass1",
	})
	if err != nil {
		t.Fatalf("Register RPC error = %v", err)
	}
	if resp.User.Username != "handleruser" {
		t.Errorf("resp.User.Username = %q, want %q", resp.User.Username, "handleruser")
	}
	if resp.AccessToken == "" {
		t.Error("resp.AccessToken is empty")
	}
}

func TestLoginHandler_Success(t *testing.T) {
	client, cleanup := newTestServer(t)
	defer cleanup()

	client.Register(context.Background(), &userpb.RegisterRequest{
		Username: "loginhandler", Password: "ValidPass1",
	})
	resp, err := client.Login(context.Background(), &userpb.LoginRequest{
		Username: "loginhandler", Password: "ValidPass1", DeviceId: "dev-001",
	})
	if err != nil {
		t.Fatalf("Login RPC error = %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("resp.AccessToken is empty")
	}
}

func TestRegisterHandler_WeakPassword(t *testing.T) {
	client, cleanup := newTestServer(t)
	defer cleanup()

	_, err := client.Register(context.Background(), &userpb.RegisterRequest{
		Username: "weak",
		Password: "short",
	})
	if err == nil {
		t.Fatal("Register RPC expected error for weak password")
	}
}
