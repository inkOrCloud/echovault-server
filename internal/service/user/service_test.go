package user_test

import (
	"errors"
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/user"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"testing"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	name := "file:user_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	resp, err := svc.Register(ctx, "newuser", "ValidPass1", "New User")
	require.NoError(t, err)
	require.NotEmpty(t, resp.UserID)
	require.Equal(t, "newuser", resp.Username)
	require.NotEmpty(t, resp.Token)
}

func TestRegister_DuplicateUsername(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	_, err := svc.Register(ctx, "dupuser", "ValidPass1", "User 1")
	require.NoError(t, err)
	_, err = svc.Register(ctx, "dupuser", "ValidPass2", "User 2")
	require.Error(t, err)
}

func TestRegister_WeakPassword(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
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
			t.Parallel()
			_, err := svc.Register(ctx, "user_"+tt.name, tt.password, "Test")
			require.Error(t, err)
		})
	}
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	_, err := svc.Register(ctx, "loginuser", "MyPass123", "Login User")
	require.NoError(t, err)
	resp, err := svc.Login(ctx, "loginuser", "MyPass123", "device-001")
	require.NoError(t, err)
	require.NotEmpty(t, resp.Token)
	require.NotEmpty(t, resp.UserID)
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	_, err := svc.Register(ctx, "userx", "CorrectPass1", "User X")
	require.NoError(t, err)
	_, err = svc.Login(ctx, "userx", "WrongPass1", "device-002")
	require.Error(t, err)
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")

	_, err := svc.Login(context.Background(), "nouser", "SomePass1", "device-003")
	require.Error(t, err)
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	regResp, err := svc.Register(ctx, "getme", "GetMePass1", "Get Me")
	require.NoError(t, err)
	u, err := svc.GetUser(ctx, regResp.UserID)
	require.NoError(t, err)
	require.Equal(t, "getme", u.Username)
}

func TestRegisterDevice_Success(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	reg, err := svc.Register(ctx, "deviceuser", "DevicePass1", "Device User")
	require.NoError(t, err)
	err = svc.RegisterDevice(ctx, reg.UserID, "dev-001", "My Desktop", "linux")
	require.NoError(t, err)
}

func TestRegisterDevice_DuplicateID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	reg, err := svc.Register(ctx, "dupdev", "DupDevPass1", "Dup Device")
	require.NoError(t, err)
	err = svc.RegisterDevice(ctx, reg.UserID, "dev-001", "First", "linux")
	require.NoError(t, err)
	err = svc.RegisterDevice(ctx, reg.UserID, "dev-001", "Second", "macos")
	require.Error(t, err)
}

func TestListDevices(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	reg, err := svc.Register(ctx, "listdev", "ListDevP1", "List Dev")
	require.NoError(t, err)
	_ = svc.RegisterDevice(ctx, reg.UserID, "d1", "Desktop", "linux")
	_ = svc.RegisterDevice(ctx, reg.UserID, "d2", "Phone", "android")

	devices, err := svc.ListDevices(ctx, reg.UserID)
	require.NoError(t, err)
	require.Len(t, devices, 2)
}

func TestRemoveDevice(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := user.NewService(client, "test-secret")
	ctx := context.Background()

	reg, err := svc.Register(ctx, "rmdev", "RmDevPass1", "Rm Dev")
	require.NoError(t, err)
	_ = svc.RegisterDevice(ctx, reg.UserID, "d1", "Desktop", "linux")
	err = svc.RemoveDevice(ctx, reg.UserID, "d1")
	require.NoError(t, err)

	devices, _ := svc.ListDevices(ctx, reg.UserID)
	require.Empty(t, devices)
}

func TestValidatePassword_TooLong(t *testing.T) {
	t.Parallel()
	err := user.ValidatePassword("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA1")
	if !errors.Is(err, user.ErrPasswordTooLong) { t.Errorf("err=%v", err) }
}
func TestGetUser_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := user.NewService(client, "s")
	_, err := svc.GetUser(context.Background(), "x")
	if !errors.Is(err, user.ErrUserNotFound) { t.Errorf("err=%v", err) }
}
func TestRemoveDevice_NotFound(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := user.NewService(client, "s")
	err := svc.RemoveDevice(context.Background(), "u", "x")
	if !errors.Is(err, user.ErrDeviceNotFound) { t.Errorf("err=%v", err) }
}
func TestUserEntToProto(t *testing.T) {
	t.Parallel()
	pb := user.EntToProto(&ent.User{ID:"u1",Username:"test",DisplayName:"Test User",Role:"user"})
	if pb.GetId() != "u1" { t.Errorf("id=%q",pb.GetId()) }
	if pb.GetUsername() != "test" { t.Errorf("user=%q",pb.GetUsername()) }
}
func TestUserEntToProto_Nil(t *testing.T) {
	t.Parallel()
	if pb := user.EntToProto(nil); pb != nil { t.Error("should be nil") }
}
func TestEntDeviceToProto(t *testing.T) {
	t.Parallel(); 
	d := &ent.Device{DeviceID:"d1",DeviceName:"Desktop",Platform:"linux"}
	pb := user.EntDeviceToProto(d)
	if pb.GetDeviceId() != "d1" { t.Errorf("id=%q",pb.GetDeviceId()) }
	if pb.GetDeviceName() != "Desktop" { t.Errorf("name=%q",pb.GetDeviceName()) }
}
func TestEntDeviceToProto_Nil(t *testing.T) {
	t.Parallel()
	if pb := user.EntDeviceToProto(nil); pb != nil { t.Error("should be nil") }
}
