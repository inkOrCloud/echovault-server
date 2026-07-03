package grpc_test

import (
	"context"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func newSyncTestServer(t *testing.T) (syncpb.SyncServiceClient, func()) {
	t.Helper()
	name := "file:sync_handler_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	svc := sync.NewService(client)
	handler := evgrpc.NewSyncHandler(svc)
	s := grpc.NewServer()
	syncpb.RegisterSyncServiceServer(s, handler)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c := syncpb.NewSyncServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestSyncPushHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newSyncTestServer(t)
	defer cleanup()

	resp, err := c.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: testDeviceID,
		Changes: []*syncpb.SyncChange{{
			EntityType: "song",
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			DeviceId:   testDeviceID,
		}},
	})
	if err != nil {
		t.Fatalf("PushChanges RPC error = %v", err)
	}
	if resp.GetServerVersion() != 1 {
		t.Errorf("ServerVersion = %d, want 1", resp.GetServerVersion())
	}
}
