package user_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return client
}

func TestRegister_Success(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	resp, err := svc.Register(ctx, "newuser", "ValidPass1", "New User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.UserID == "" {
		t.Error("Register() returned empty UserID")
	}
	if resp.Username != "newuser" {
		t.Errorf("Register() Username = %q, want %q", resp.Username, "newuser")
	}
	if resp.Token == "" {
		t.Error("Register() returned empty Token")
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	svc.Register(ctx, "dupuser", "ValidPass1", "User 1")
	_, err := svc.Register(ctx, "dupuser", "ValidPass2", "User 2")
	if err == nil {
		t.Fatal("Register() expected error for duplicate username")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Ab1"},
		{"has space", "pass word123"},
		{"has tab", "pass\tword123"},
		{"has newline", "pass\nword123"},
		{"non-ASCII", "passwörd123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, "user_"+tt.name, tt.password, "Test")
			if err == nil {
				t.Errorf("Register() with password %q expected error", tt.password)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	svc.Register(ctx, "loginuser", "MyPass123", "Login User")
	resp, err := svc.Login(ctx, "loginuser", "MyPass123", "device-001")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.Token == "" {
		t.Error("Login() returned empty token")
	}
	if resp.UserID == "" {
		t.Error("Login() returned empty UserID")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	svc.Register(ctx, "userx", "CorrectPass1", "User X")
	_, err := svc.Login(ctx, "userx", "WrongPass1", "device-002")
	if err == nil {
		t.Fatal("Login() expected error for wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")

	_, err := svc.Login(context.Background(), "nouser", "SomePass1", "device-003")
	if err == nil {
		t.Fatal("Login() expected error for nonexistent user")
	}
}

func TestGetUser_Success(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	regResp, _ := svc.Register(ctx, "getme", "GetMePass1", "Get Me")
	u, err := svc.GetUser(ctx, regResp.UserID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if u.Username != "getme" {
		t.Errorf("GetUser() Username = %q, want %q", u.Username, "getme")
	}
}
